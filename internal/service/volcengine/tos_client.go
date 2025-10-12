package volcengine

import (
	"fmt"
	"context"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"

	"github.com/gogf/gf/v2/frame/g"
)

func init() {
	fmt.Println("TOS GO SDK Version: ", tos.Version)
	ctx := context.Background()

	ak := g.Cfg().MustGet(ctx, "volc.tos.ak").String()
	sk := g.Cfg().MustGet(ctx, "volc.tos.sk").String()

	credential := tos.NewStaticCredentials(ak, sk)

}