package transcription

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"

	v1 "doubao-speech-service/api/transcription/v1"
	"doubao-speech-service/internal/dao"
)

func (c *ControllerV1) Search(ctx context.Context, req *v1.SearchReq) (res *v1.SearchRes, err error) {
	res = &v1.SearchRes{}
	userID := g.RequestFromCtx(ctx).Header.Get("X-User-ID")
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))

	cols := dao.Transcription.Columns()
	condition := fmt.Sprintf(
		"(LOWER(%s) LIKE ? OR LOWER(%s) LIKE ? OR LOWER(%s) LIKE ?)",
		cols.TaskId, cols.RequestId, cols.Status,
	)

	if err = dao.Transcription.Ctx(ctx).
		Where(cols.Owner+" = ?", userID).
		Where(condition, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		OrderDesc(cols.Id).
		Scan(res); err != nil {
		return nil, gerror.Wrap(err, "查询数据库失败")
	}
	return res, nil
}
