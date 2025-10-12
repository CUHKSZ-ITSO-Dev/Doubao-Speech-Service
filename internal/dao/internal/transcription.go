// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2025-10-12 13:10:28
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// TranscriptionDao is the data access object for the table transcription.
type TranscriptionDao struct {
	table    string               // table is the underlying table name of the DAO.
	group    string               // group is the database configuration group name of the current DAO.
	columns  TranscriptionColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler   // handlers for customized model modification.
}

// TranscriptionColumns defines and stores column names for the table transcription.
type TranscriptionColumns struct {
	Id                        string //
	TaskId                    string //
	RequestId                 string //
	UploadParams              string //
	Status                    string //
	AudioTranscriptionFile    string //
	ChapterFile               string //
	InformationExtractionFile string //
	SummarizationFile         string //
	TranslationFile           string //
	LastQueryAt               string //
	UpdatedAt                 string //
	CreatedAt                 string //
}

// transcriptionColumns holds the columns for the table transcription.
var transcriptionColumns = TranscriptionColumns{
	Id:                        "id",
	TaskId:                    "task_id",
	RequestId:                 "request_id",
	UploadParams:              "upload_params",
	Status:                    "status",
	AudioTranscriptionFile:    "audio_transcription_file",
	ChapterFile:               "chapter_file",
	InformationExtractionFile: "information_extraction_file",
	SummarizationFile:         "summarization_file",
	TranslationFile:           "translation_file",
	LastQueryAt:               "last_query_at",
	UpdatedAt:                 "updated_at",
	CreatedAt:                 "created_at",
}

// NewTranscriptionDao creates and returns a new DAO object for table data access.
func NewTranscriptionDao(handlers ...gdb.ModelHandler) *TranscriptionDao {
	return &TranscriptionDao{
		group:    "default",
		table:    "transcription",
		columns:  transcriptionColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *TranscriptionDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *TranscriptionDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *TranscriptionDao) Columns() TranscriptionColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *TranscriptionDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *TranscriptionDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *TranscriptionDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
