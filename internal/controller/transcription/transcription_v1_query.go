package transcription

import (
	"context"

	"github.com/gogf/gf/v2/errors/gerror"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/dao"
)

func (c *ControllerV1) Query(ctx context.Context, req *v1.QueryReq) (res *v1.QueryRes, err error) {
	if err = dao.Transcription.Ctx(ctx).
		Where("owner = ?", req.Owner).
		Scan(res); err != nil {
		return nil, gerror.Wrap(err, "查询数据库失败")
	}
	return
}
