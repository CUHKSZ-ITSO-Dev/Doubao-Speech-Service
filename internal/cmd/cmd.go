package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"doubao-speech-service/internal/controller/transcription"
	"doubao-speech-service/internal/middlewares"
	transcriptionSvc "doubao-speech-service/internal/service/transcription"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			s := g.Server()
			s.SetPort(g.Cfg().MustGet(ctx, "server.port").Int())
			s.SetClientMaxBodySize(1024 * 1024 * 1024)
			s.Use(middlewares.BrotliMiddleware)

			oai := s.GetOpenApi()
			oai.Config.CommonResponse = ghttp.DefaultHandlerResponse{}
			oai.Config.CommonResponseDataField = "data"
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
