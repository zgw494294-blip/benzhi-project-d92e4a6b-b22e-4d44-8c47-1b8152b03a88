# BENZHI_README

基于 Go 实现的洁净工作台现场认证与启用放行服务 HTTP API 项目，一款后端服务，已完整实现洁净工作台现场认证与启用放行服务，通过版本化 JSON HTTP API 串联设备建档、方案锁定、有序实测、自动偏差、整改复测、质量冻结、凭据签发与公开校验，并以摘要链账本和原子投影保证可追溯恢复。

## 项目说明
- 项目：benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88
- 项目用途：已完整实现洁净工作台现场认证与启用放行服务，通过版本化 JSON HTTP API 串联设备建档、方案锁定、有序实测、自动偏差、整改复测、质量冻结、凭据签发与公开校验，并以摘要链账本和原子投影保证可追溯恢复。
- Go 工具链：`golang:1.22`
- 前端工具链：无

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88-arm64 linux/arm64
docker run -it benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -selfcheck -addr=127.0.0.1:19081`
