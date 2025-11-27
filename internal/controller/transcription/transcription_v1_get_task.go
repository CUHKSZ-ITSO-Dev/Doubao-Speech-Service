package transcription

import (
	"context"

	"github.com/gogf/gf/v2/errors/gerror"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/dao"
)

func (c *ControllerV1) GetTask(ctx context.Context, req *v1.GetTaskReq) (res *v1.GetTaskRes, err error) {
	res = &v1.GetTaskRes{}
	transcriptionRecord, err := dao.Transcription.Ctx(ctx).Where("request_id = ?", req.RequestId).One()
	if err != nil {
		return nil, gerror.Wrap(err, "查询任务失败")
	}
	if transcriptionRecord.IsEmpty() {
		return nil, gerror.New("任务不存在")
	}
	if err = transcriptionRecord.Struct(res); err != nil {
		return nil, gerror.Wrap(err, "解析任务数据失败")
	}
	return res, nil
}
