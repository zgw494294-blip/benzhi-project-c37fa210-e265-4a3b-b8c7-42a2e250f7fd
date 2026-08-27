# BENZHI_README

基于 Go 实现的stagecaption-finalizer Web 项目，一款后端服务，剧场字幕译稿定稿工作台，覆盖术语约束、字幕修订、确定性校验、问题整改、独立复核、冻结导出和完整性核验。

## 项目说明
- 项目：benzhi-project-c37fa210-e265-4a3b-b8c7-42a2e250f7fd
- 项目用途：剧场字幕译稿定稿工作台，覆盖术语约束、字幕修订、确定性校验、问题整改、独立复核、冻结导出和完整性核验。
- Go 工具链：`golang:1.22`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-c37fa210-e265-4a3b-b8c7-42a2e250f7fd-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-c37fa210-e265-4a3b-b8c7-42a2e250f7fd-arm64 linux/arm64
docker run -it benzhi-project-c37fa210-e265-4a3b-b8c7-42a2e250f7fd-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -selfcheck -addr=127.0.0.1:19081`
