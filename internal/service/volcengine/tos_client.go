package volcengine

import (
	"context"
	"fmt"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"

	"github.com/gogf/gf/v2/frame/g"
)

var client *tos.ClientV2

func init() {
	fmt.Println("TOS GO SDK Version: ", tos.Version)
	ctx := context.Background()

	credential := tos.NewStaticCredentials(g.Cfg().MustGet(ctx, "volc.tos.ak").String(), g.Cfg().MustGet(ctx, "volc.tos.sk").String())
	var err error
	client, err = tos.NewClientV2(
		g.Cfg().MustGet(ctx, "volc.tos.endpoint").String(),
		tos.WithCredentials(credential),
		tos.WithRegion(g.Cfg().MustGet(ctx, "volc.tos.region").String()),
	)

	if err != nil {
		fmt.Println("Error:", err)
		panic(err)
	}
}

func GetClient() *tos.ClientV2 {
	return client
}
