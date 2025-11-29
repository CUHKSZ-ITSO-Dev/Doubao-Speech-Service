package transcription

import (
	"context"

	"github.com/gogf/gf/v2/errors/gerror"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/dao"
)

func (c *ControllerV1) GetTask(ctx context.Context, req *v1.GetTaskReq) (res *v1.GetTaskRes, err error) {
	res = &v1.GetTaskRes{}
	err = dao.Transcription.Ctx(ctx).Where("request_id = ?", req.RequestId).Scan(&res)
	if err != nil {
		err = gerror.Wrap(err, "获取任务记录失败")
	}
	return
}
