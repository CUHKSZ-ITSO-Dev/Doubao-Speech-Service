package transcription

import (
	"context"

	v1 "doubao-speech-service/api/transcription/v1"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
)

func (c *ControllerV1) Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error) {
	requestId := uuid.New().String()
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
			"https://openspeech.bytedance.com/api/v3/auc/lark/submit",
			req,
		); 
		
	if resVar.IsEmpty() {
		return nil, gerror.Wrap(err, "提交任务失败")
	} 
	if err = resVar.Scan(&res); err != nil || res.Data.TaskID == "" {
		return nil, gerror.Wrap(err, "返回结果为空")
	}
	res.Data.RequestID = requestId
	return
}
