package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gctx"

	"github.com/golang-migrate/migrate/v4"
	// 导入数据库驱动
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// 导入迁移文件源
	_ "github.com/golang-migrate/migrate/v4/source/file"

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

			// 将 GoFrame 格式转换为标准的 PostgreSQL 连接字符串
			gfLink := g.Cfg().MustGet(ctx, "database.default.link").String()
			pgLink := convertGoFrameLinkToPostgres(gfLink)

			m, err := migrate.New(
				"file://./manifest/migrations/postgres",
				pgLink,
			)
			if err != nil {
				log.Fatalf("创建migrate实例失败: %v", err)
			}

			// 执行所有升级迁移
			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				log.Fatalf("执行迁移失败: %v", err)
			}

			// 获取当前版本
			version, dirty, err := m.Version()
			if err != nil && err != migrate.ErrNilVersion {
				log.Fatalf("获取版本失败: %v", err)
			}

			fmt.Printf("当前数据库版本: %d, 是否处于脏状态: %t\n", version, dirty)

			s := g.Server()
			if err := meetingRecordSvc.Init(ctx); err != nil {
				g.Log().Warningf(ctx, "meeting record service init failed: %v", err)
			}
			s.SetPort(g.Cfg().MustGet(ctx, "server.port").Int())
			s.SetClientMaxBodySize(1024 * 1024 * 1024)
			s.Use(middlewares.BrotliMiddleware)
			s.Use(ghttp.MiddlewareCORS)
			s = setupWebSocketHandler(ctx, s)
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

func setupWebSocketHandler(ctx context.Context, s *ghttp.Server) *ghttp.Server {
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
		g.Log().Infof(ctx, "r.Header: %v", r.Header)
		traceID := gctx.CtxId(ctx)
		userID := r.Header.Get("X-User-ID")
		r.Response.Write(traceID)

		if userID == "" {
			r.Response.WriteJson(g.Map{
				"code":    http.StatusUnauthorized,
				"message": "userID is required",
			})
			return
		}

		sessionID := r.Session.MustId()
		protocol := "http"
		if r.TLS != nil {
			protocol = "https"
		}
		g.Log().Infof(ctx, `[微服务-请求] request=%+v url=%s session=%s user=%s remote=%s ua=%s protocol=%s`, r.Request, r.URL.String(), sessionID, userID, r.RemoteAddr, r.Header.Get("User-Agent"), protocol)

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
			g.Log().Infof(ctx, "火山引擎连接已建立，connect_id=%s, logid=%s", traceID, logID)
		} else {
			g.Log().Infof(ctx, "火山引擎连接已建立，connect_id=%s", traceID)
		}

		recorder, err := meetingRecordSvc.NewRecorder(ctx, traceID)
		if err != nil && !errors.Is(err, meetingRecordSvc.ErrRecorderDisabled) {
			g.Log().Warningf(ctx, "录音初始化失败，connect_id=%s: %v", traceID, err)
		}

		errCh := make(chan error, 2)

		go meetingRecordSvc.ProxyWebSocket(ctx, "client", clientConn, upstreamConn, recorder, errCh)
		go meetingRecordSvc.ProxyWebSocket(ctx, "upstream", upstreamConn, clientConn, nil, errCh)

		g.Log().Infof(ctx, "开始会话")

		firstErr := <-errCh
		_ = clientConn.Close()
		_ = upstreamConn.Close()
		secondErr := <-errCh

		if !isNormalClosure(firstErr) {
			g.Log().Warningf(ctx, "WebSocket 转发通道异常关闭 (first): %v", firstErr)
		}
		if !isNormalClosure(secondErr) {
			g.Log().Warningf(ctx, "WebSocket 转发通道异常关闭 (second): %v", secondErr)
		}

		if recorder != nil {
			// g.Log().Infof(reqCtx, "开始收尾录音，connect_id=%s", connectID)
			if result, err := recorder.Finalize(); err != nil {
				g.Log().Warningf(ctx, "录音收尾失败，connect_id=%s: %v", traceID, err)
			} else if result != nil {
				g.Log().Infof(ctx, "录音完成，connect_id=%s, bytes=%d", traceID, result.Size)
				result.Owner = userID
				meetingRecordSvc.EnqueueUpload(ctx, result)
			}
		}

		g.Log().Infof(ctx, "WebSocket 转发完成，connect_id=%s", traceID)
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

// convertGoFrameLinkToPostgres 将 GoFrame 格式的数据库连接字符串转换为标准的 PostgreSQL 连接字符串
// GoFrame 格式: pgsql:username:password@tcp(host:port)/dbname?params
// 标准格式: postgres://username:password@host:port/dbname?params
func convertGoFrameLinkToPostgres(gfLink string) string {
	// 如果已经是标准格式，直接返回
	if strings.HasPrefix(gfLink, "postgres://") || strings.HasPrefix(gfLink, "postgresql://") {
		return gfLink
	}

	// 解析 GoFrame 格式: pgsql:username:password@tcp(host:port)/dbname?params
	// 正则表达式匹配: type:username:password@protocol(host:port)/dbname?params
	re := regexp.MustCompile(`^(\w+):([^:]+):([^@]+)@(\w+)\(([^)]+)\)/([^?]+)(\?.*)?$`)
	matches := re.FindStringSubmatch(gfLink)
	if len(matches) != 8 {
		// 如果解析失败，尝试直接使用（可能是其他格式）
		return gfLink
	}

	dbType := matches[1]
	username := matches[2]
	password := matches[3]
	_ = matches[4] // protocol (tcp), not used
	hostPort := matches[5]
	dbname := matches[6]
	params := matches[7]

	// 只处理 pgsql 类型
	if dbType != "pgsql" && dbType != "postgres" {
		return gfLink
	}

	// 忽略协议（tcp），直接使用 host:port
	// 构建标准 PostgreSQL 连接字符串
	result := fmt.Sprintf("postgres://%s:%s@%s/%s%s", username, password, hostPort, dbname, params)
	return result
}
