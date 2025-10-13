package transcription

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

func Polling(taskId, requestId string) {
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
		defer cancel()
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()

		for range t.C {
			status, err := Query(bgCtx, taskId, requestId)
			if err != nil || status == "running" || status == "pending" {
				continue
			}
			g.Log().Infof(bgCtx, "[%s] 任务 %s 轮询结束。最终状态：%s", requestId, taskId, status)
			break
		}
	}()
}
