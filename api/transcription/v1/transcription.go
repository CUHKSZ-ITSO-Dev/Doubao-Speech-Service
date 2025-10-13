package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"doubao-speech-service/internal/model/entity"
)

// 文件上传API（支持单文件和多文件）
type FileUploadReq struct {
	g.Meta `path:"/file/upload" method:"post" summary:"文件上传"`
	// multipart/form-data. Key = files
}
type FileUploadRes struct {
	Files   []FileInfo  `json:"files" dc:"成功上传的文件列表"`
	Errors  []FileError `json:"errors,omitempty" dc:"上传失败的文件错误信息"`
	Total   int         `json:"total" dc:"总文件数"`
	Success int         `json:"success" dc:"成功上传数"`
	Failed  int         `json:"failed" dc:"上传失败数"`
}
type FileInfo struct {
	FileID   string `json:"file_id" dc:"文件唯一标识"`
	FileURL  string `json:"file_url" dc:"文件访问地址"`
	FileType string `json:"file_type" dc:"文件类型"`
	FileSize int64  `json:"file_size" dc:"文件大小(字节)"`
	FileName string `json:"file_name" dc:"文件名称"`
}
type FileError struct {
	FileName string `json:"file_name" dc:"文件名"`
	Error    string `json:"error" dc:"错误信息"`
}

// 任务提交API
type TaskSubmitReq struct {
	g.Meta `path:"/task/submit" method:"post" summary:"任务提交"`
	FileID string           `json:"FileID" v:"required" dc:"文件ID，通过文件上传API获得"`
	Params TaskSubmitParams `json:"Params" v:"required" dc:"任务参数"`
}
type TaskSubmitRes struct {
	TaskID    string `json:"task_id" dc:"任务ID"`
	RequestID string `json:"request_id" dc:"请求ID"`
	Status    string `json:"status" dc:"任务状态"`
}
type TaskSubmitParams struct {
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
}

type ListReq struct {
	g.Meta `path:"/list" method:"post" summary:"任务查询"`
	Owner  string `v:"required" dc:"所有者"`
}
type ListRes []entity.Transcription
