-- 更新 transcription 表，添加文件相关字段

ALTER TABLE transcription ADD COLUMN file_url TEXT;
ALTER TABLE transcription ADD COLUMN file_type VARCHAR(10);  -- 'audio' or 'video'
ALTER TABLE transcription ADD COLUMN file_size BIGINT;

-- 更新索引
CREATE INDEX idx_transcription_request_id ON transcription(request_id);
CREATE INDEX idx_transcription_status ON transcription(task_id);
CREATE INDEX idx_transcription_status ON transcription(status);

-- 历史数据兼容性处理（如果需要）
UPDATE transcription SET status = 'completed' WHERE status IS NULL AND task_id IS NOT NULL;
