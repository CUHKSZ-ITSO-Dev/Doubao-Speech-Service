package transcription

import (
	"context"
	"sync"
	"time"

	"doubao-speech-service/internal/consts"
	"doubao-speech-service/internal/dao"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
)

type QueryRes struct {
	Code    int    `json:"Code" v:"required" dc:"状态码 0: 成功"`
	Message string `json:"Message" v:"required" dc:"状态"`
	Data    struct {
		TaskID string `v:"required" json:"TaskID" dc:"任务ID"`
		Status string
		Result struct {
			AudioTranscriptionFile    string `json:"AudioTranscriptionFile"`
			ChapterFile               string `json:"ChapterFile"`
			InformationExtractionFile string `json:"InformationExtractionFile"`
			SummarizationFile         string `json:"SummarizationFile"`
			TranslationFile           string `json:"TranslationFile"`
		} `json:"Result"`
	}
}

type FetchResult struct {
	Key    string
	Result *gjson.Json
}

func Query(ctx context.Context, taskId string, requestId string) (string, error) {
	r, err := g.Client().Timeout(5*time.Second).ContentJson().
		SetHeaderMap(g.MapStrStr{
			"X-Api-App-Key":     g.Cfg().MustGet(ctx, "volc.lark.appid").String(),
			"X-Api-Access-Key":  g.Cfg().MustGet(ctx, "volc.lark.accessKey").String(),
			"X-Api-Resource-Id": "volc.lark.minutes",
			"X-Api-Request-Id":  requestId,
			"X-Api-Sequence":    "-1",
		}).
		Post(
			ctx,
			"https://openspeech.bytedance.com/api/v3/auc/lark/query",
			g.Map{
				"TaskID": taskId,
			},
		)
	if err != nil {
		return "", gerror.New("向第三方服务器发送查询请求失败")
	}
	defer r.Close()

	bodyStr := r.ReadAllString()
	if r.Response.Header.Get("X-Api-Message") != "OK" {
		statusCode := r.Response.Header.Get("X-Api-Status-Code")
		logid := r.Response.Header.Get("X-Tt-Logid")
		bodyPreview := bodyStr
		if len(bodyPreview) > 500 {
			bodyPreview = gstr.SubStr(bodyPreview, 0, 500) + "..."
		}
		g.Log().Errorf(ctx, "[%s] 任务 %s 查询失败。StatusCode=%s Message=%s Mapped=%s Logid=%s Body=%s",
			requestId,
			taskId,
			statusCode,
			r.Response.Header.Get("X-Api-Message"),
			consts.GetErrMsg(ctx, statusCode),
			logid,
			bodyPreview,
		)
		return "", gerror.Newf(
			"第三方服务返回非OK。StatusCode=%s Message=%s Logid=%s",
			statusCode,
			r.Response.Header.Get("X-Api-Message"),
			logid,
		)
	}

	var queryRes *QueryRes
	if err = gconv.Struct(bodyStr, &queryRes); err != nil {
		return "", gerror.Wrap(err, "返回结果格式化失败")
	}
	if queryRes.Data.Status != "success" {
		dao.Transcription.Ctx(ctx).Data(g.Map{
			"status": queryRes.Data.Status,
		}).Where("task_id = ? and request_id = ?", taskId, requestId).Update()
	} else {
		// 成功了
		var wg sync.WaitGroup
		results := make(chan *FetchResult, 5)
		client := g.Client()

		tasks := g.MapStrStr{
			"audio_transcription_file":    queryRes.Data.Result.AudioTranscriptionFile,
			"chapter_file":                queryRes.Data.Result.ChapterFile,
			"information_extraction_file": queryRes.Data.Result.InformationExtractionFile,
			"summarization_file":          queryRes.Data.Result.SummarizationFile,
			"translation_file":            queryRes.Data.Result.TranslationFile,
		}

		for key, url := range tasks {
			if url != "" {
				wg.Add(1)
				go func(k, u string) {
					defer wg.Done()
					results <- &FetchResult{
						Key:    k,
						Result: gjson.New(client.GetContent(ctx, url)),
					}
				}(key, url)
			}
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		updateData := g.Map{}
		for res := range results {
			updateData[res.Key] = res.Result
		}
		updateData["status"] = queryRes.Data.Status
		if _, err = dao.Transcription.Ctx(ctx).
			Data(updateData).
			Where("task_id = ? and request_id = ?", taskId, requestId).Update(); err != nil {
			return "", gerror.Wrap(err, "更新数据库失败")
		}
	}

	g.Log().Infof(ctx, "[%s] 任务 %s 查询结果：%s", requestId, taskId, queryRes.Data.Status)
	return queryRes.Data.Status, nil
}
