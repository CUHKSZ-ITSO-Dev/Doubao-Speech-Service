package volcengine

import (
	"github.com/gogf/gf/v2/frame/g"

	"github.com/volcengine/volcengine-go-sdk/service/billing"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

func consoleInit() {
	g.Log().Info(ctx, "Volcengine Console SDK Version:", volcengine.SDKVersion)
	config := volcengine.NewConfig().
		WithRegion(g.Cfg().MustGet(ctx, "volc.region").String()).
		WithCredentials(credentials.NewStaticCredentials(
			g.Cfg().MustGet(ctx, "volc.ak").String(),
			g.Cfg().MustGet(ctx, "volc.sk").String(),
			"",
		))
	sess, err := session.NewSession(config)
	if err != nil {
		panic(err)
	}
	_ = billing.New(sess)
	// listBillDetailInput := &billing.ListBillDetailInput{
	// 	BillPeriod: volcengine.String("2025-09"),
	// 	Limit:      volcengine.Int32(20),
	// }
	// bills, err := svc.ListBillDetail(listBillDetailInput)
	// if err != nil {
	// 	panic(err)
	// }
	// g.Dump(bills)
	g.Log().Info(ctx, "Volcengine Console SDK initialized")
}

