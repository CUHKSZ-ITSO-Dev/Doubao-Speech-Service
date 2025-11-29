package transcription

import (
	"context"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/model/entity"
	"doubao-speech-service/internal/service/volcengine"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

func (c *ControllerV1) GetFileURL(ctx context.Context, req *v1.GetFileURLReq) (res *v1.GetFileURLRes, err error) {
	res = &v1.GetFileURLRes{}
	owner := g.RequestFromCtx(ctx).Header.Get("X-User-ID")

	var transRecord *entity.Transcription
	if err := dao.Transcription.Ctx(ctx).Where("request_id = ?", req.RequestId).Where("owner = ?", owner).Limit(1).Scan(&transRecord); err != nil {
		return nil, gerror.Wrap(err, "查询任务记录失败")
	}
	fileURL, err := volcengine.GetFileURL(ctx, transRecord)
	if err != nil {
		return nil, gerror.Wrap(err, "获取文件URL失败")
	}
	res.FileURL = fileURL
	return res, nil
}
