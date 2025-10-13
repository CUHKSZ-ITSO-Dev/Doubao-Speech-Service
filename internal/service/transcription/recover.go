package transcription

import (
	"context"
	"time"

	"doubao-speech-service/internal/dao"
	"doubao-speech-service/internal/model/entity"

	"github.com/gogf/gf/v2/frame/g"
)

func Recover(ctx context.Context) {
	transRecords := []entity.Transcription{}
	dao.Transcription.Ctx(ctx).WhereIn("status", []string{"pending", "running"}).Scan(&transRecords)
	for _, v := range transRecords {
		go func(taskId, requestId string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
			defer cancel()
			t := time.NewTicker(30 * time.Second)
			defer t.Stop()
			for range t.C {
				status, err := Query(bgCtx, taskId, requestId)
				if err != nil || status == "running" || status == "pending" {
					continue
				}
				g.Log().Infof(bgCtx, "[%s] 任务 %s 查询结束。最终状态：%s", requestId, taskId, status)
				break
			}
		}(v.TaskId, v.RequestId)
	}
	if len(transRecords) > 0 {
		g.Log().Infof(ctx, "已恢复 pending 状态任务 %d 个", len(transRecords))
	}
}
