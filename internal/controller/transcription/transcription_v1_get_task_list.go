package transcription

import (
	"context"
	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/dao"
	"fmt"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

func (c *ControllerV1) GetTaskList(ctx context.Context, req *v1.GetTaskListReq) (res *v1.GetTaskListRes, err error) {
	res = &v1.GetTaskListRes{}
	userID := g.RequestFromCtx(ctx).Header.Get("X-User-ID")

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	cols := dao.Transcription.Columns()

	model := dao.Transcription.Ctx(ctx).
		Where(cols.Owner+" = ?", userID)

	if req.LastRequestID != "" && req.LastRequestID != "0" {
		var anchor struct {
			Id        int64       `json:"id"`
			CreatedAt *gtime.Time `json:"created_at"`
		}

		if err = dao.Transcription.Ctx(ctx).
			Fields(cols.Id, cols.CreatedAt).
			Where(cols.Owner+" = ?", userID).
			Where(cols.RequestId+" = ?", req.LastRequestID).
			Limit(1).
			Scan(&anchor); err != nil {
			return nil, gerror.Wrap(err, "查询数据库失败")
		}

		if anchor.Id == 0 || anchor.CreatedAt == nil {
			return nil, gerror.New("last_request_id不存在或无效")
		}

		model = model.Where(
			fmt.Sprintf("(%s < ?) OR (%s = ? AND %s < ?)", cols.CreatedAt, cols.CreatedAt, cols.Id),
			anchor.CreatedAt, anchor.CreatedAt, anchor.Id,
		)
	}

	if err = model.
		OrderDesc(cols.CreatedAt).
		OrderDesc(cols.Id).
		Limit(limit).
		Scan(&res.TaskMetas); err != nil {
		return nil, gerror.Wrap(err, "查询数据库失败")
	}
	res.Total, err = model.Count()
	return res, nil
}
