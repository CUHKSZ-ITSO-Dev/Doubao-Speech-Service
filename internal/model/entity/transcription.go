// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2025-10-12 19:49:44
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/encoding/gjson"
)

// Transcription is the golang structure for table transcription.
type Transcription struct {
	Id                        int         `json:"id"                        orm:"id"                          description:""` //
	TaskId                    string      `json:"taskId"                    orm:"task_id"                     description:""` //
	RequestId                 string      `json:"requestId"                 orm:"request_id"                  description:""` //
	UploadParams              *gjson.Json `json:"uploadParams"              orm:"upload_params"               description:""` //
	Status                    string      `json:"status"                    orm:"status"                      description:""` //
	AudioTranscriptionFile    *gjson.Json `json:"audioTranscriptionFile"    orm:"audio_transcription_file"    description:""` //
	ChapterFile               *gjson.Json `json:"chapterFile"               orm:"chapter_file"                description:""` //
	InformationExtractionFile *gjson.Json `json:"informationExtractionFile" orm:"information_extraction_file" description:""` //
	SummarizationFile         *gjson.Json `json:"summarizationFile"         orm:"summarization_file"          description:""` //
	TranslationFile           *gjson.Json `json:"translationFile"           orm:"translation_file"            description:""` //
	UpdatedAt                 string      `json:"updatedAt"                 orm:"updated_at"                  description:""` //
	CreatedAt                 string      `json:"createdAt"                 orm:"created_at"                  description:""` //
}
