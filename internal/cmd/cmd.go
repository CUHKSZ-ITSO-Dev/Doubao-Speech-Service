package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"doubao-speech-service/internal/controller/transcription"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			s := g.Server()
			s.SetPort(g.Cfg().MustGet(ctx, "server.port").Int())
			s.SetOpenApiPath(g.Cfg().MustGet(ctx, "server.openapiPath").String())
			s.SetSwaggerPath(g.Cfg().MustGet(ctx, "server.swaggerPath").String())
			s.SetClientMaxBodySize(1024 * 1024 * 1024)

			s.Group("/transcription", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					transcription.NewV1(),
				)
			})
			s.Run()
			return nil
		},
	}
)
