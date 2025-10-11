package transcription

import (
	"context"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"

	v1 "doubao-speech-service/api/transcription/v1"
)

func (c *ControllerV1) Query(ctx context.Context, req *v1.QueryReq) (res *v1.QueryRes, err error) {
	client, err := g.Client().ContentJson().
		SetHeaderMap(g.MapStrStr{
			"X-Api-App-Key":     g.Cfg().MustGet(ctx, "volc-lark-minutes.appid").String(),
			"X-Api-Access-Key":  g.Cfg().MustGet(ctx, "volc-lark-minutes.accessKey").String(),
			"X-Api-Resource-Id": "volc.lark.minutes",
			"X-Api-Request-Id":  g.RequestFromCtx(ctx).GetHeader("X-Api-Request-Id"),
			"X-Api-Sequence":    "-1",
		}).
		Post(
			ctx,
			"https://openspeech.bytedance.com/api/v3/auc/lark/query",
			req,
		)
	if err != nil {
		return nil, gerror.Wrap(err, "提交任务失败")
	}
	defer client.Close()

	// client.RawDump()

	if gconv.Struct(client.ReadAllString(), &res); err != nil || res.Data.TaskID == "" {
		return nil, gerror.Wrap(err, "任务查询失败")
	}
	return
}
