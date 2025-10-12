package transcription

import (
	"context"
	"doubao-speech-service/internal/consts"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
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

func Query(ctx context.Context, taskId string, requestId string) (string, error) {
	r, err := g.Client().ContentJson().
		SetHeaderMap(g.MapStrStr{
			"X-Api-App-Key":     g.Cfg().MustGet(ctx, "volc-lark-minutes.appid").String(),
			"X-Api-Access-Key":  g.Cfg().MustGet(ctx, "volc-lark-minutes.accessKey").String(),
			"X-Api-Resource-Id": "volc.lark.minutes",
			"X-Api-Request-Id":  requestId,
			"X-Api-Sequence":    "-1",
		}).
		Post(
			ctx,
			"https://openspeech.bytedance.com/api/v3/auc/lark/query",
			taskId,
		)
	if err != nil {
		return "", gerror.New("向第三方服务器发送查询请求失败")
	}
	defer r.Close()

	if r.Response.Header.Get("X-Api-Message") != "OK" {
		return "", gerror.Newf(
			"第三方服务通知任务处理失败。错误码：%s，错误信息：%s。Logid：%s",
			r.Response.Header.Get("X-Api-Error-Message"),
			consts.GetErrMsg(r.Response.Header.Get("X-Api-Error-Message")),
			r.Response.Header.Get("X-Tt-Logid"),
		)
	}

	var queryRes *QueryRes
	if err = gconv.Struct(r.ReadAllString(), &queryRes); err != nil {
		return "", gerror.Wrap(err, "返回结果格式化失败")
	}

	return queryRes.Data.Status, nil
}
