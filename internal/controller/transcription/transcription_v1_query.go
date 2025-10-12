package transcription

import (
	"context"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/service/transcription"
)

func (c *ControllerV1) Query(ctx context.Context, req *v1.QueryReq) (res *v1.QueryRes, err error) {
	resVar, err := transcription.Query(ctx, req.TaskID, g.RequestFromCtx(ctx).GetHeader("X-Api-Request-Id"))
	if err != nil {
		return nil, gerror.Wrap(err, "调用查询服务失败")
	}
	if err = resVar.Scan(&res); err != nil || res.Data.TaskID == "" {
		return nil, gerror.Wrap(err, "无法处理返回结果，或者返回结果为空")
	}
	return
}
