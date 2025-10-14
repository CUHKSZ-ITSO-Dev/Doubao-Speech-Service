package volcengine

import (
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"

	"github.com/gogf/gf/v2/frame/g"
)

var client *tos.ClientV2

func tosInit() {
	g.Log().Info(ctx, "Volcengine TOS GO SDK Version:", tos.Version)

	credential := tos.NewStaticCredentials(g.Cfg().MustGet(ctx, "volc.ak").String(), g.Cfg().MustGet(ctx, "volc.sk").String())
	var err error
	if client, err = tos.NewClientV2(
		g.Cfg().MustGet(ctx, "volc.tos.endpoint").String(),
		tos.WithCredentials(credential),
		tos.WithRegion(g.Cfg().MustGet(ctx, "volc.tos.region").String()),
	); err != nil {
		panic(err)
	}
	g.Log().Info(ctx, "Volcengine TOS Client initialized")
}

func GetClient() *tos.ClientV2 {
	return client
}
