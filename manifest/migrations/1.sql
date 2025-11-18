CREATE TABLE IF NOT EXISTS transcription (
    id SERIAL PRIMARY KEY,
    task_id TEXT,
    request_id TEXT NOT NULL,
    owner TEXT NOT NULL,
    file_info JSONB,
    status TEXT NOT NULL,
    task_params JSONB,
    audio_transcription_file JSONB,
    chapter_file JSONB,
    information_extraction_file JSONB,
    summarization_file JSONB,
    translation_file JSONB,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transcription_request_id ON transcription(request_id);
CREATE INDEX idx_transcription_task_id ON transcription(task_id);
CREATE INDEX idx_transcription_status ON transcription(status);
CREATE INDEX idx_transcription_owner ON transcription(owner);