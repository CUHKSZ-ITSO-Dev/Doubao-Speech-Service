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
	keyword := strings.TrimSpace(req.Keyword)
	if keyword == "" {
		return nil, gerror.New("关键词不能为空")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	keyword = strings.ToLower(keyword)

	cols := dao.Transcription.Columns()
	condition := fmt.Sprintf(
		"(LOWER(%s) LIKE ? OR LOWER(%s) LIKE ? OR LOWER(%s) LIKE ? OR LOWER(COALESCE(%s->>'filename', '')) LIKE ?)",
		cols.RequestId, cols.Status, cols.FileInfo,
	)

	if err = dao.Transcription.Ctx(ctx).
		Where(cols.Owner+" = ?", userID).
		Where(condition, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		OrderDesc(cols.CreatedAt).
		OrderDesc(cols.Id).
		Limit(limit).
		Scan(res); err != nil {
		return nil, gerror.Wrap(err, "查询数据库失败")
	}
	return res, nil
}
