package consts

import "github.com/gogf/gf/v2/frame/g"

var (
	errMsg = g.MapStrStr{
		"20000000": "成功",
		"20000001": "正在处理中",
		"20000002": "任务在队列中",
		"20000003": "静音音频，返回该错误码无需重新query，直接重新submit",
		"45000001": "请求参数无效，请求参数缺失必需字段 / 字段值无效 / 重复请求。",
		"45000002": "空音频",
		"45000151": "音频格式不正确",
		"550xxxxx": "服务内部处理错误",
		"55000031": "服务器繁忙，服务过载，无法处理当前请求。",
	}

	TranscriptionExt = g.MapStrStr{
		".mp4":  "video",
		".avi":  "video",
		".mov":  "video",
		".mkv":  "video",
		".wmv":  "video",
		".flv":  "video",
		".mp3":  "audio",
		".wav":  "audio",
		".aac":  "audio",
		".flac": "audio",
		".ogg":  "audio",
	}
)

const (
	MaxUploadSize = 1024 * 1024 * 1024 // 1GB
)

func GetErrMsg(code string) string {
	msg, ok := errMsg[code]
	if !ok && code[:2] == "550" {
		msg = errMsg["550xxxxx"]
	}
	return msg
}
