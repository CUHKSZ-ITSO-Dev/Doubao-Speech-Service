package transcription

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/errors/gerror"
)

func Query(ctx context.Context, taskId string, requestId string) (*g.Var, error) {
	resVar := g.Client().ContentJson().
		SetHeaderMap(g.MapStrStr{
			"X-Api-App-Key":     g.Cfg().MustGet(ctx, "volc-lark-minutes.appid").String(),
			"X-Api-Access-Key":  g.Cfg().MustGet(ctx, "volc-lark-minutes.accessKey").String(),
			"X-Api-Resource-Id": "volc.lark.minutes",
			"X-Api-Request-Id":  requestId,
			"X-Api-Sequence":    "-1",
		}).
		PostVar(
			ctx,
			"https://openspeech.bytedance.com/api/v3/auc/lark/query",
			taskId,
		)
	if resVar.IsEmpty() {
		return nil, gerror.New("向第三方服务器请求失败")
	}
	return resVar, nil
}