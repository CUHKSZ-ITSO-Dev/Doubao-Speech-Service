# 豆包语音服务 - 豆包语音妙记服务

- 豆包妙记大模型文档：https://www.volcengine.com/docs/6561/1798094
- Nick Wilde 在 Apifox 邀请你加入项目 豆包语音服务 https://app.apifox.com/invite/project?token=gIlr1vIJZXKNDHVkmIKgR （七天有效期）
- Apifox 里面的接口 “成功” 接口示例里面有数据，也可以直接用。

## 架构设计

### 阶段一：pending - /upload
1. 调用 /transcription/file/upload, 上传文件，multipart/form-data，key：files。文件校验完成后，会在数据库里生成一条记录，生成一个requestID，并把状态改成 pending。
2. 目前也没有本地存储文件的机制。因此pending状态其实目前没什么用，也没法重试上传到公有云。

### 阶段二：uploaded - /upload
1. 文件校验完成后就开始上传文件到火山云 TOS。上传完成后，并生成下载直链之后，会把数据库状态改成 uploaded，把直链写入同一条数据库记录。此时 /upload 控制器完成。

### 阶段三：submitting - /submit
1. 调用 /transcription/task/submit，提交 requestID 和 处理参数。后端成功提交任务到火山云之后，就会把返回的 TaskID，提交的处理参数写入同一条数据库记录，并把状态改成 submitted。

### 阶段四：running - Polling Goroutines
1. /submit 控制器在最后会启动 Polling goroutines 开始轮询服务器处理状态。同时/submit 控制器会结束请求并返回给前端 TaskID。

### 阶段五：success：成功 failed：失败 - Polling Goroutines
1. 当 Polling goroutines 每半分钟的轮询（最多24小时）后，如果收到云返回的状态为success 或者 failed：
	- failed：将数据库记录状态改成 failed。同时 goroutine 终止。
	- success：成功了，则会返回每个模块功能内容下载的链接。开 # 个 goroutine 并发请求数据。然后把每个获取到的结果存入数据库对应字段，并把状态改成 success。同时 goroutine 终止。
2. 如果收到 running 或者查询时出错，则忽略，继续等待下次轮询。

### 查询接口：/list
1. 传入 owner，返回所有的会议纪要记录。
2. 因为现在还没有用户系统的接入，owner在upload的环节已经被硬编码为 test@test。

### 其他接口：内部服务 Recover
1. 后端启动时会扫描一遍数据库。对于状态为 submitted 和 running 的记录，每个记录开启一个 Polling goroutine 进行轮询。同时会有日志数据显示恢复了 x 个任务。

## 后端启动：

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
