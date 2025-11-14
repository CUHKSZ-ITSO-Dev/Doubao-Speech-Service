CREATE TABLE IF NOT EXISTS transcription (
    id SERIAL PRIMARY KEY,
    task_id TEXT,
    request_id TEXT NOT NULL,
    owner TEXT NOT NULL,
    file_info JSON,
    status TEXT NOT NULL,
    task_params JSON,
    audio_transcription_file JSON,
    chapter_file JSON,
    information_extraction_file JSON,
    summarization_file JSON,
    translation_file JSON,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transcription_request_id ON transcription(request_id);
CREATE INDEX idx_transcription_task_id ON transcription(task_id);
CREATE INDEX idx_transcription_status ON transcription(status);
CREATE INDEX idx_transcription_owner ON transcription(owner);