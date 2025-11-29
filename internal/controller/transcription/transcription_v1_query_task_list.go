package transcription

import (
	"context"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/dao"
)

func (c *ControllerV1) QueryTaskList(ctx context.Context, req *v1.QueryTaskListReq) (res *v1.QueryTaskListRes, err error) {
	res = &v1.QueryTaskListRes{}
	userID := g.RequestFromCtx(ctx).Header.Get("X-User-ID")
	if len(req.RequestIDs) > 100 {
		return nil, gerror.New("请求ID数量超过限制：最多100个")
	}

	cols := dao.Transcription.Columns()
	if err = dao.Transcription.Ctx(ctx).
		Where(cols.Owner+" = ?", userID).
		WhereIn(cols.RequestId, req.RequestIDs).
		Scan(&res.TaskMetas); err != nil {
		return nil, gerror.Wrap(err, "查询数据库失败")
	}
	return res, nil
}
