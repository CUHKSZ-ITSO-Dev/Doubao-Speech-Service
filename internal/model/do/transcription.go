// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2025-10-13 18:49:11
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Transcription is the golang structure of table transcription for DAO operations like Where/Data.
type Transcription struct {
	g.Meta                    `orm:"table:transcription, do:true"`
	Id                        any         //
	TaskId                    any         //
	RequestId                 any         //
	Owner                     any         //
	FileInfo                  *gjson.Json //
	Status                    any         //
	TaskParams                *gjson.Json //
	AudioTranscriptionFile    *gjson.Json //
	ChapterFile               *gjson.Json //
	InformationExtractionFile *gjson.Json //
	SummarizationFile         *gjson.Json //
	TranslationFile           *gjson.Json //
	UpdatedAt                 *gtime.Time //
	CreatedAt                 *gtime.Time //
}
