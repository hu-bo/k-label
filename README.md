# Kline Vector Platform

Enterprise-grade stack: Go-Kratos gRPC backend + Qdrant vector DB + Ant Design Pro frontend. Supports docker-compose one-command deploy.


## 🚀 快速启动


访问: http://localhost:8080

## 📋 接口验证

```powershell
# 新增向量
curl -X POST http://localhost:8080/api/vectors -H "Content-Type: application/json" -d '{
  "close": 111, "ma_15m": 10, "ma_4h": 20, "ma_1d": 30, "rsi": 46
}'

# 查询列表
curl http://localhost:8080/api/vectors?limit=10

# 相似度搜索
curl -X POST http://localhost:8080/api/vectors/search -H "Content-Type: application/json" -d '{
  "query": {"close": 110, "ma_15m": 12, "ma_4h": 22, "ma_1d": 32, "rsi": 48},
  "top_k": 5
}'
```

## 📁 项目结构


## 🛠️ 代码生成与依赖

### 1. 安装依赖

- Go 相关：
  ```powershell
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
  go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
  ```
- 安装 protoc（https://github.com/protocolbuffers/protobuf/releases）

### 2. 生成 Go gRPC 及 HTTP 网关代码

在项目根目录执行：

```powershell
protoc -I api/vector/v1 \
  --go_out=paths=source_relative:api/vector/v1 \
  --go-grpc_out=paths=source_relative:api/vector/v1 \
  --grpc-gateway_out=paths=source_relative,grpc_api_configuration=api/vector/v1/vector.yaml:api/vector/v1 \
  api/vector/v1/vector.proto
```

- 生成的 Go 代码和 HTTP 网关代码会在 `api/vector/v1/` 目录下。
- 可选：生成 OpenAPI 文档

```powershell
protoc -I api/vector/v1 \
  --openapiv2_out=api/vector/v1 \
  api/vector/v1/vector.proto
```

### 3. 依赖包

- `google.golang.org/grpc`
- `github.com/grpc-ecosystem/grpc-gateway/v2`

如需自动化脚本，可将上述命令写入 Makefile 或 PowerShell 脚本。

## 🔄 后续优化 (可选)

1. **Proto生成**: 
2. **生产配置**: 环境变量 + 配置文件
3. **前端打包**: `npm run build`优化
4. **部署**: Docker化 + K8s编排

**✨ 当前状态**: 功能完整，可直接运行和测试。前后端接口已对接，向量逻辑完备。
