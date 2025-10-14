# 豆包语音服务 - 豆包语音妙记服务

- 豆包妙记大模型文档：https://www.volcengine.com/docs/6561/1798094
- Nick Wilde 在 Apifox 邀请你加入项目 豆包语音服务 https://app.apifox.com/invite/project?token=gIlr1vIJZXKNDHVkmIKgR （七天有效期）
  Apifox 里面的接口 “成功” 接口示例里面有数据，也可以直接用。

## 启动：

1. 需要一个 PostgreSQL 数据库
	- 端口：5431
	- 用户：doubao
	- 密码：doubao
	- 数据库：doubao-speech-service
2. 数据库迁移脚本 （manifest/migrations/1.sql）
```sql
CREATE TABLE IF NOT EXISTS transcription (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
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
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_transcription_request_id ON transcription(request_id);
CREATE INDEX idx_transcription_task_id ON transcription(task_id);
CREATE INDEX idx_transcription_status ON transcription(status);
CREATE INDEX idx_transcription_owner ON transcription(owner);
```
3. 火山云的相关 Key 已经封装在二进制文件里面
4. 后端服务在 8500 端口上启动
