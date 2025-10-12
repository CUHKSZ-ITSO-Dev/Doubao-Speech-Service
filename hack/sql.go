package main

import (
	"database/sql"
	"log"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	// connect
	db, err := sql.Open("sqlite", "/Users/nickwilde/Repo/Doubao-Speech-Service/db.sqlite3")
	if err != nil {
		log.Fatal(err)
	}

	// get SQLite version
	_, _ = db.Query(`
		CREATE TABLE IF NOT EXISTS transcription (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT NOT NULL,
			request_id TEXT NOT NULL,
			upload_params JSON,
			status TEXT NOT NULL,
			audio_transcription_file JSON,
			chapter_file JSON,
			information_extraction_file JSON,
			summarization_file JSON,
			translation_file JSON,
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)

}
