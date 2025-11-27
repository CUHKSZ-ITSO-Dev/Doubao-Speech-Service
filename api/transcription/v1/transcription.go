package v1

import (
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// 文件上传API（支持单文件和多文件）
type UploadFileReq struct {
	g.Meta `path:"/file/upload" method:"post" summary:"上传文件" dc:"使用 multipart/form-data 方式上传（可批量，并行处理）。字段名是 files。"`
}
type UploadFileRes struct {
	TaskMetas []TaskMeta  `json:"taskMetas" dc:"成功上传的任务元数据列表"`
	Errors    []FileError `json:"errors,omitempty" dc:"上传失败的文件错误信息"`
	Total     int         `json:"total" dc:"总文件数"`
	Success   int         `json:"success" dc:"成功上传数"`
	Failed    int         `json:"failed" dc:"上传失败数"`
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
	g.Meta    `path:"/task/submit" method:"post" summary:"提交任务"`
	RequestId string           `json:"RequestId" v:"required" dc:"请求ID，通过文件上传API获得"`
	Params    TaskSubmitParams `json:"Params" v:"required" dc:"任务参数"`
}
type TaskSubmitRes struct {
	Status string `json:"status" dc:"任务状态"`
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

type TaskMeta struct {
	RequestId  string      `json:"requestId" dc:"请求 ID"`
	Owner      string      `json:"owner" dc:"拥有者 UPN"`
	FileInfo   *gjson.Json `json:"fileInfo" dc:"文件信息"`
	Status     string      `json:"status" dc:"任务状态"`
	TaskParams *gjson.Json `json:"taskParams" dc:"任务参数"`
	CreatedAt  *gtime.Time `json:"createdAt" dc:"创建时间"`
}

type Task struct {
	TaskMeta
	AudioTranscriptionFile    *gjson.Json `json:"audioTranscriptionFile" dc:"语音转写信息"`
	ChapterFile               *gjson.Json `json:"chapterFile" dc:"章节总结信息"`
	InformationExtractionFile *gjson.Json `json:"informationExtractionFile" dc:"信息提取信息"`
	SummarizationFile         *gjson.Json `json:"summarizationFile" dc:"全文总结信息"`
	TranslationFile           *gjson.Json `json:"translationFile" dc:"翻译信息"`
}

type GetTaskListReq struct {
	g.Meta        `path:"/list" method:"get" resEg:"resource/interface/transcription/get_task_list_res.json" summary:"获取任务列表"`
	LastRequestID string `json:"last_request_id" d:"0" dc:"当前列表最后一条数据的RequestID，用于基于该RequestID向后分页"`
	Limit         int    `json:"limit" d:"10" v:"min:1|max:100" dc:"本次请求返回的数据条数"`
}

type GetTaskListRes struct {
	Total     int        `json:"total" dc:"总条目数"`
	TaskMetas []TaskMeta `json:"taskMetas" dc:"任务列表"`
}

type SearchReq struct {
	g.Meta  `path:"/search" method:"get" summary:"搜索任务"`
	Keyword string `json:"keyword" v:"required" dc:"关键词"`
	Limit   int    `json:"limit" d:"20" v:"min:1|max:100" dc:"返回条数，默认20，最大100"`
}

type SearchRes []TaskMeta

type GetTaskReq struct {
	g.Meta    `path:"/task/{request_id}" resEg:"resource/interface/transcription/get_task_res.json" method:"get" summary:"获取任务详情"`
	RequestId string `json:"request_id" v:"required" dc:"请求ID"`
}

type GetTaskRes Task

type DeleteTaskReq struct {
	g.Meta    `path:"/task/{request_id}" method:"delete" summary:"删除任务"`
	RequestId string `json:"request_id" v:"required" dc:"请求ID"`
}
type DeleteTaskRes struct {
	Success bool `json:"success" dc:"是否删除成功"`
}

type QueryTaskListReq struct {
	g.Meta     `path:"/task/query" method:"get" summary:"批量查询任务"`
	RequestIDs []string `json:"request_ids" v:"required" dc:"请求ID列表"`
}

type QueryTaskListRes struct {
	TaskMetas []TaskMeta `json:"taskMetas" dc:"任务元数据列表"`
}

type GetFileURLReq struct {
	g.Meta    `path:"/task/{request_id}/file" method:"get" summary:"获取文件URL"`
	RequestId string `json:"request_id" v:"required" dc:"请求ID"`
}

type GetFileURLRes struct {
	FileURL string `json:"file_url" dc:"文件URL"`
}
