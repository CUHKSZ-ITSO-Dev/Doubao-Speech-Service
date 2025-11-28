package transcription

import (
	"context"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/dao"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

func (c *ControllerV1) DeleteTask(ctx context.Context, req *v1.DeleteTaskReq) (res *v1.DeleteTaskRes, err error) {
	res = &v1.DeleteTaskRes{}
	if sqlRes, err := dao.Transcription.Ctx(ctx).Where("request_id = ?", req.RequestId).Delete(); err != nil {
		return nil, gerror.WrapCode(gcode.CodeDbOperationError, err, "删除任务失败")
	} else if eftRow, err := sqlRes.RowsAffected(); eftRow == 0 {
		return nil, gerror.New("找不到任务。数据库影响行数为0。")
	} else if err != nil {
		return nil, gerror.WrapCode(gcode.CodeDbOperationError, err, "检查任务删除情况失败")
	}
	res.Success = true
	return
}
