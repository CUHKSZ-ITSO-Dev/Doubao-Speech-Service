-- 启用 trigram 支持，用于不区分大小写的子字符串搜索
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 为 request_id 创建 trigram 索引
CREATE INDEX IF NOT EXISTS idx_transcription_request_id_trgm ON transcription USING GIN (request_id gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_transcription_filename_trgm ON transcription USING GIN (COALESCE(file_info->>'filename', '') gin_trgm_ops);
