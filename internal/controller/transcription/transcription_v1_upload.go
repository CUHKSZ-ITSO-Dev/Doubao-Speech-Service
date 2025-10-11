package transcription

import (
	"context"

	v1 "doubao-speech-service/api/transcription/v1"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/google/uuid"
)

func (c *ControllerV1) Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error) {
	client, err := g.Client().ContentJson().
		SetHeaderMap(g.MapStrStr{
			"X-Api-App-Key":     g.Cfg().MustGet(ctx, "volc-lark-minutes.appid").String(),
			"X-Api-Access-Key":  g.Cfg().MustGet(ctx, "volc-lark-minutes.accessKey").String(),
			"X-Api-Resource-Id": "volc.lark.minutes",
			"X-Api-Request-Id":  uuid.New().String(),
			"X-Api-Sequence":    "-1",
		}).
		Post(
			ctx,
			"https://openspeech.bytedance.com/api/v3/auc/lark/submit",
			req,
		)
	if err != nil {
		return nil, gerror.Wrap(err, "提交任务失败")
	}
	defer func() {
		client.RawDump()
		_ = client.Close()
	}()

	if gconv.Struct(client.ReadAllString(), &res); err != nil || res.Data.TaskID == "" {
		return nil, gerror.Wrap(err, "提交任务失败")
	}
	return
}
