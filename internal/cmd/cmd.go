package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/glog"
	"github.com/google/uuid"
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
			logger := g.Log()
			if err := meetingRecordSvc.Init(ctx); err != nil {
				logger.Warningf(ctx, "meeting record service init failed: %v", err)
			}
			s.SetPort(g.Cfg().MustGet(ctx, "server.port").Int())
			s.SetClientMaxBodySize(1024 * 1024 * 1024)
			s.Use(middlewares.BrotliMiddleware)
			s.Use(ghttp.MiddlewareCORS)
			s = setupWebSocketHandler(s, logger)
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

func setupWebSocketHandler(s *ghttp.Server, logger *glog.Logger) *ghttp.Server {
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
		// Upgrade HTTP connection to WebSocket
		connectID := uuid.NewString()
		r.Response.Header().Set("X-Api-Connect-Id", connectID)

		clientConn, err := wsUpGrader.Upgrade(r.Response.Writer, r.Request, nil)
		if err != nil {
			r.Response.Write(err.Error())
			return
		}
		defer clientConn.Close()

		reqCtx := r.Context()

		endpoint := g.Cfg().MustGet(reqCtx, "volc.asr.endpoint").String()
		appKey := g.Cfg().MustGet(reqCtx, "volc.asr.appKey").String()
		accessKey := g.Cfg().MustGet(reqCtx, "volc.asr.accessKey").String()
		resourceID := g.Cfg().MustGet(reqCtx, "volc.asr.resourceId").String()

		if endpoint == "" || appKey == "" || accessKey == "" || resourceID == "" {
			logger.Error(reqCtx, "WebSocket 转发所需的火山引擎配置缺失，请检查 volc.asr.* 配置")
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
		upstreamHeaders.Set("X-Api-Connect-Id", connectID)

		upstreamConn, resp, err := dialer.DialContext(reqCtx, endpoint, upstreamHeaders)
		if err != nil {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			logger.Errorf(reqCtx, "连接火山引擎双向流式识别服务失败: %v", err)
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
			logger.Infof(reqCtx, "火山引擎连接已建立，connect_id=%s, logid=%s", connectID, logID)
		} else {
			logger.Infof(reqCtx, "火山引擎连接已建立，connect_id=%s", connectID)
		}

		recorder, err := meetingRecordSvc.NewRecorder(reqCtx, connectID)
		if err != nil && !errors.Is(err, meetingRecordSvc.ErrRecorderDisabled) {
			logger.Warningf(reqCtx, "录音初始化失败，connect_id=%s: %v", connectID, err)
		}

		errCh := make(chan error, 2)

		go meetingRecordSvc.ProxyWebSocket(reqCtx, "client", clientConn, upstreamConn, recorder, errCh)
		go meetingRecordSvc.ProxyWebSocket(reqCtx, "upstream", upstreamConn, clientConn, nil, errCh)

		logger.Infof(reqCtx, "开始会话")

		firstErr := <-errCh
		_ = clientConn.Close()
		_ = upstreamConn.Close()
		secondErr := <-errCh

		if !isNormalClosure(firstErr) {
			logger.Warningf(reqCtx, "WebSocket 转发通道异常关闭 (first): %v", firstErr)
		}
		if !isNormalClosure(secondErr) {
			logger.Warningf(reqCtx, "WebSocket 转发通道异常关闭 (second): %v", secondErr)
		}

		if recorder != nil {
			// logger.Infof(reqCtx, "开始收尾录音，connect_id=%s", connectID)
			if result, err := recorder.Finalize(); err != nil {
				logger.Warningf(reqCtx, "录音收尾失败，connect_id=%s: %v", connectID, err)
			} else if result != nil {
				logger.Infof(reqCtx, "录音完成，connect_id=%s, bytes=%d", connectID, result.Size)
				meetingRecordSvc.EnqueueUpload(reqCtx, result)
			}
		}

		logger.Infof(reqCtx, "WebSocket 转发完成，connect_id=%s", connectID)
	})

	// // Configure static file serving
	// s.SetServerRoot("static")
	return s
}

func isNormalClosure(err error) bool {
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived, websocket.CloseGoingAway) {
		return true
	}
	return false
}
