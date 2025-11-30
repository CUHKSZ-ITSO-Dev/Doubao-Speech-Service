package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gorilla/websocket"

	"doubao-speech-service/internal/controller/transcription"
	"doubao-speech-service/internal/middlewares"
	meetingRecordSvc "doubao-speech-service/internal/service/meetingRecord"
	transcriptionSvc "doubao-speech-service/internal/service/transcription"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			fmt.Println(`
 ____                       _       ____                  _
/ ___| _ __   ___  ___  ___| |__   / ___|  ___ _ ____   _(_) ___ ___
\___ \| '_ \ / _ \/ _ \/ __| '_ \  \___ \ / _ \ '__\ \ / / |/ __/ _ \
 ___) | |_) |  __/  __/ (__| | | |  ___) |  __/ |   \ V /| | (_|  __/
|____/| .__/ \___|\___|\___|_| |_| |____/ \___|_|    \_/ |_|\___\___|
      |_|
					 `)
			fmt.Println("Doubao Speech Microservice")
			fmt.Println("Copyright 2025 The Chinese University of Hong Kong, Shenzhen")
			fmt.Println()
			s := g.Server()
			s.SetPort(g.Cfg().MustGet(ctx, "server.port").Int())
			s.SetClientMaxBodySize(1024 * 1024 * 1024)
			s.Use(middlewares.BrotliMiddleware)
			s.Use(ghttp.MiddlewareCORS)
			s = setupWebSocketHandler(s)
			oai := s.GetOpenApi()
			oai.Config.CommonResponse = ghttp.DefaultHandlerResponse{}
			oai.Config.CommonResponseDataField = "Data"
			s.SetOpenApiPath(g.Cfg().MustGet(ctx, "server.openapiPath").String())
			s.SetSwaggerPath(g.Cfg().MustGet(ctx, "server.swaggerPath").String())

			s.Group("/transcription", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					transcription.NewV1(),
				)
			})

			go transcriptionSvc.Recover(ctx)

			s.Run()
			return nil
		},
	}
)

func setupWebSocketHandler(s *ghttp.Server) *ghttp.Server {
	var (
		wsUpGrader = websocket.Upgrader{
			// TODO: 同源检查
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			},
		}
	)

	// Bind WebSocket handler to /ws endpoint
	dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: true,
	}

	s.BindHandler("/doubao-speech-service/ws", func(r *ghttp.Request) {
		ctx := r.Context()
		// Upgrade HTTP connection to WebSocket
		userID := r.Header.Get("X-User-ID")

		if userID == "" {
			g.Log().Warningf(ctx, "Unauthorized request, userID is required")
			r.Response.WriteJson(g.Map{
				"code":    http.StatusUnauthorized,
				"message": "userID is required",
			})
			return
		}

		// 处理 WebSocket 升级
		clientConn, err := wsUpGrader.Upgrade(r.Response.Writer, r.Request, nil)
		if err != nil {
			r.Response.Write(err.Error())
			return
		}
		defer clientConn.Close()

		endpoint := g.Cfg().MustGet(ctx, "volc.asr.endpoint").String()
		appKey := g.Cfg().MustGet(ctx, "volc.asr.appKey").String()
		accessKey := g.Cfg().MustGet(ctx, "volc.asr.accessKey").String()
		resourceID := g.Cfg().MustGet(ctx, "volc.asr.resourceId").String()

		if endpoint == "" || appKey == "" || accessKey == "" || resourceID == "" {
			g.Log().Error(ctx, "WebSocket 转发所需的火山引擎配置缺失，请检查 volc.asr.* 配置")
			_ = clientConn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "server configuration error"),
				time.Now().Add(time.Second),
			)
			return
		}

		upstreamHeaders := http.Header{}
		upstreamHeaders.Set("X-Api-App-Key", appKey)
		upstreamHeaders.Set("X-Api-Access-Key", accessKey)
		upstreamHeaders.Set("X-Api-Resource-Id", resourceID)

		upstreamConn, resp, err := dialer.DialContext(ctx, endpoint, upstreamHeaders)
		if err != nil {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			g.Log().Errorf(ctx, "连接火山引擎双向流式识别服务失败: %v", err)
			_ = clientConn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "upstream unavailable"),
				time.Now().Add(time.Second),
			)
			return
		}
		defer upstreamConn.Close()

		logID := ""
		if resp != nil {
			logID = resp.Header.Get("X-Tt-Logid")
		}
		if logID != "" {
			g.Log().Infof(ctx, "火山引擎连接已建立，logid=%s", logID)
		} else {
			g.Log().Infof(ctx, "火山引擎连接已建立，未返回 logID")
		}

		// 初始化录音机
		recorder, err := meetingRecordSvc.NewRecorder(ctx)
		if err != nil && !meetingRecordSvc.IsRecorderDisabled(err) {
			g.Log().Error(ctx, gerror.Wrap(err, "录音初始化失败"))
			_ = clientConn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "录音器初始化失败"),
				time.Now().Add(time.Second),
			)
			return
		}

		// 两个 WebSocket Proxy 启动。同时启动录音完成善后处理 goroutine。
		// 因为善后处理完成之后，需要给 client 发送 task-complete 消息。
		// 因此善后处理完成后才会开始关闭两个 WebSocket Proxy。
		errCh := make(chan *meetingRecordSvc.WsErrorMessage, 2)
		taskCompleteCh := make(chan *meetingRecordSvc.RecordingResult, 1)
		taskCompleteNotifyCh := make(chan *meetingRecordSvc.RecordingResult, 1)
		serverFinalReceivedCh := make(chan struct{}, 1)
		go meetingRecordSvc.ProxyWebSocket(ctx, "client -> bytedance", clientConn, upstreamConn, recorder, errCh, nil, nil)
		go meetingRecordSvc.ProxyWebSocket(ctx, "bytedance -> client", upstreamConn, clientConn, nil, errCh, taskCompleteNotifyCh, serverFinalReceivedCh)
		go handleFinalize(ctx, recorder, taskCompleteCh, taskCompleteNotifyCh, serverFinalReceivedCh)

		// 阻塞，等待错误消息。
		hasError := false
		if errMsg := <-errCh; !isNormalClosure(errMsg.Err) {
			g.Log().Warning(ctx, gerror.Wrapf(errMsg.Err, "WebSocket 转发通道异常关闭: %s", errMsg.Source))
			hasError = true
		}
		if errMsg := <-errCh; !isNormalClosure(errMsg.Err) {
			g.Log().Warning(ctx, gerror.Wrapf(errMsg.Err, "WebSocket 转发通道异常关闭: %s", errMsg.Source))
			hasError = true
		}
		if hasError {
			_ = clientConn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "WebSocket 双向转发通道发生异常错误"),
				time.Now().Add(time.Second),
			)
			g.Log().Error(ctx, "WebSocket 转发通道返回错误，录音过程发生了错误。请参考上方 Warning 警告 “WebSocket 转发通道异常关闭” 的报错信息。")
			return
		}

		// 阻塞，等待善后工作完成
		// taskCompleteCh 由 handleFinalize 填充，handleFinalize 完成之后，会关闭 taskCompleteCh
		// 所以如果 taskCompleteCh 有消息了，说明转换什么的都好了。
		if result := <-taskCompleteCh; result.ConnectID != "-1" {
			g.Log().Infof(ctx, "将录音加入上传队列")
			meetingRecordSvc.EnqueueUpload(ctx, result)
		} else {
			g.Log().Errorf(ctx, "音频善后 Finalize 失败：%s", result.FilePath)
		}
	})

	return s
}

func isNormalClosure(err error) bool {
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived, websocket.CloseGoingAway)
}

func handleFinalize(ctx context.Context, recorder *meetingRecordSvc.Recorder, taskCompleteCh chan *meetingRecordSvc.RecordingResult, taskCompleteNotifyCh chan *meetingRecordSvc.RecordingResult, serverFinalReceivedCh chan struct{}) {
	defer close(taskCompleteCh)
	defer close(taskCompleteNotifyCh)
	<-serverFinalReceivedCh

	g.Log().Infof(ctx, "开始处理录音")
	if result, err := recorder.Finalize(); err != nil {
		g.Log().Error(ctx, gerror.Wrap(err, "录音收尾失败"))
		// 临时用 RecordingResult 的这三个参数传递错误信息
		taskCompleteCh <- &meetingRecordSvc.RecordingResult{
			ConnectID: "-1",
			Owner:     g.RequestFromCtx(ctx).Header.Get("X-User-ID"),
			FilePath:  "Error: " + err.Error(),
		}
	} else if result != nil {
		g.Log().Infof(ctx, "录音处理完成，bytes=%d", result.Size)
		result.Owner = g.RequestFromCtx(ctx).Header.Get("X-User-ID")
		// TODO： taskCompleteNotifych 什么时候发
		taskCompleteCh <- result
		taskCompleteNotifyCh <- result
	} else {
		g.Log().Error(ctx, "录音处理完成，但没有结果 result == nil")
		taskCompleteCh <- &meetingRecordSvc.RecordingResult{
			ConnectID: "-1",
			Owner:     g.RequestFromCtx(ctx).Header.Get("X-User-ID"),
			FilePath:  "Error: 录音处理完成，但没有结果 result == nil",
		}
	}
}
