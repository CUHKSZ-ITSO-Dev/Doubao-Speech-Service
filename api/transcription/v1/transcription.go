package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/encoding/gjson"

	"doubao-speech-service/internal/model/entity"
)

type UploadReq struct {
	g.Meta `path:"/upload" method:"post" summary:"任务创建"`
	Input  struct {
		Offline struct {
			FileURL  string `json:"FileURL" dc:"文件url 文件大小< 1G 时长2小时"`
			FileType string `json:"FileType" v:"required|in:audio,video" dc:"文件类型，audio：音频，video：视频"`
		} `json:"Offline" v:"required"`
	} `json:"Input" v:"required"`

	Params struct {
		AllActivate bool   `json:"AllActivate" v:"required" dc:"是否打包计费。[非全功能使用，具体功能需设置设对应功能属性为true]"`
		SourceLang  string `json:"SourceLang" v:"required|in:zh_cn,en_us" dc:"原始语种，zh_cn：中。en_us：英"`

		AudioTranscriptionEnable bool `json:"AudioTranscriptionEnable" v:"required" d:"true" dc:"是否开启语音转写。必须传 true"`
		AudioTranscriptionParams struct {
			SpeakerIdentification bool `json:"SpeakerIdentification" v:"required" dc:"是否开启说话人识别"`
			NumberOfSpeaker       int  `json:"NumberOfSpeaker" v:"required" d:"0" dc:"说话人数量，为0时算法自动识别。如果知道会议几个说话人可写，如果不知道默认写0"`
			NeedWordTimeSeries    bool `json:"NeedWordTimeSeries" v:"required" dc:"是否需要单词时间序列"`
		} `json:"AudioTranscriptionParams" v:"required"`

		// 附加功能1：翻译
		TranslationEnable bool `json:"TranslationEnable" dc:"是否翻译转写文本"`
		TranslationParams struct {
			TargetLang string `json:"TargetLang" v:"in:zh_cn,en_us" dc:"目标语言"`
		} `json:"TranslationParams"`

		// 附加功能2、3：代办提取 或 问答提取
		InformationExtractionEnabled bool `json:"InformationExtractionEnabled" dc:"是否需要文章结构化数据"`
		InformationExtractionParams  struct {
			Types []string `json:"Types" v:"foreach|in:todo_list,question_answer" dc:"todo_list : 待办提取。question_answer:问答提取"`
		} `json:"InformationExtractionParams"`

		// 附加功能4：全文总结
		SummarizationEnabled bool `json:"SummarizationEnabled" dc:"是否开启全文总结"`
		SummarizationParams  struct {
			Types []string `json:"Types" v:"foreach|in:summary" dc:"summary:全文总结"`
		} `json:"SummarizationParams"`

		// 附加功能5：章节总结
		ChapterEnabled bool `json:"ChapterEnabled" dc:"是否开启章节总结"`
	} `json:"Params"`
}
type UploadRes struct {
	Result *gjson.Json `json:"Result" dc:"结果"`
}

type QueryReq struct {
	g.Meta `path:"/query" method:"post" summary:"任务查询"`
	TaskID string `v:"required" dc:"任务ID"`
}
type QueryRes entity.Transcription
