# stagecaption-finalizer

这是面向剧场字幕译员、校审员和字幕负责人的译稿定稿工作台。系统集中管理项目语言与时码规则、项目术语约束、完整字幕修订、确定性校验、问题整改、独立复核、冻结清单以及演出字幕包导出。冻结清单带有 SHA-256 核验码，审计事件使用追加式摘要链保存，便于确认历史内容没有被静默改写。

## 构建

```text
go build ./cmd/server
```

## 运行

```text
go run ./cmd/server -addr=127.0.0.1:19081
```

浏览器打开 `http://127.0.0.1:19081/workbench`。也可以通过 `-addr=127.0.0.1:<port>` 指定回环地址，未指定时读取 `PORT` 端口号；数据默认保存在 `.stagecaption-data`。

## 工作流

负责人可在草稿或已退回阶段修改项目标题、语言、帧率和显示时长。规则变化会使当前有效校验失效，必须按新规则重新校验。译员可从工作台原子导入术语 JSON 数组；冲突响应会标出数组行号，失败批次不会写入部分条目。

每次修订保留提交人、父修订、术语版本、摘要和提交时间。工作台可查看父子修订的 cue 级差异，筛选当前问题、批量填写整改说明，并比较不同校验运行。批量关闭旧问题不会直接放行送审，仍需提交替代修订并重新校验。复核页同时显示父修订差异、校验摘要、术语版本和历史退回意见。

批准后先预览冻结清单，再以 `expectedVersion` 和 `idempotencyKey` 确认冻结。冻结项目可以分别下载 SRT 字幕、术语附录和审计记录；下载响应包含项目、清单、核验码和内容 SHA-256。完整性核验逐项检查清单码、字幕、术语、规则、批准决定、审计链、审计锚点和持久化投影。

## 主要 API

- `PUT /api/projects/{id}/rules`：修改项目规则。
- `POST /api/projects/{id}/terms/batch`：批量导入术语。
- `GET /api/projects/{id}/glossary?version={version}`：查询指定术语快照。
- `GET /api/projects/{id}/revisions`、`GET /api/projects/{id}/revisions/{revisionID}`：查询修订历史或指定修订。
- `GET /api/projects/{id}/revisions/diff?from={revisionID}&to={revisionID}`：比较两个同项目修订。
- `GET /api/projects/{id}/findings?severity=&ruleCode=&status=&cueSequence=`：筛选当前问题。
- `GET /api/projects/{id}/validation-runs?revisionID=`、`GET /api/projects/{id}/validation-runs/compare?before=&after=`：查询及比较校验运行。
- `POST /api/projects/{id}/findings/resolve-batch`：原子批量整改。
- `GET /api/projects/{id}/review-detail`：查询复核上下文及历史决定。
- `GET /api/projects/{id}/freeze/preview`：预览冻结清单。
- `GET /api/projects/{id}/exports/{captions|glossary|audit}`：分项下载冻结内容。
- `GET /api/projects/{id}/verify`：执行只读完整性核验。

## 测试

```text
go test ./...
go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
```

`-selfcheck` 会启动真实 HTTP 监听，在限定时间内通过 API 完成建档、术语、修订、校验、送审、复核、冻结、导出和核验，然后自行退出。
