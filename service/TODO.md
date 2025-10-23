# Kline Label  Platform
Enterprise-grade stack: Go-Kratos gRPC backend + Qdrant vector DB + Ant Design Pro frontend. Supports docker-compose deploy.

https://go-kratos.dev/docs/getting-started/start/
https://github.com/go-kratos/kratos
https://pro.ant.design/zh-CN/docs/folder

## 背景
将k线图数据手工打标成交易信号并编码成向量，服务端存储向量，并提供开放接口实现向量匹配,
示例数据：
```json
[
  {
    "close": 111,
    "symbol": "ETHUSDT",
    "datetime": 1761147149286,
    "lable": 1, 
    "ma_vector":[0.0237,0.0176,0.0441],
    "price_short_vector":[-0.3671,1.077,-1.657,-1.351,-0.2234,0.6229,1.673,1.405,0.9997,-0.6702,-1.227,-0.7792,0.07501,-0.1578,0.5798],
    "price_long_vector":[-2.567,-2.174,-2.329,-1.973,-1.41,-1.169,-1.289,-1.237,-1.551,-1.145,-0.2718,-0.6041,-0.8523,-0.2534,-0.5139,-0.08981,-0.3628,-1.007,0.1119,0.5201,0.2245,0.1156,0.2145,1.046,1.087,0.8687,0.6902,0.8247,1.115,1.716,1.483,1.165,1.442,1.278,1.213,1.158,0.9653,-0.1224,0.1174,-0.7764,0.1387,0.4995,0.02676,-0.5028,0.276,0.06071,0.8457,-0.6403,-0.4741,0.1388,0.5989,1.17,1.024,0.8038,-0.1041,-0.4067,-0.1633,0.3011,0.1745,0.5755]
  },
  {
    "close": 112,
    "symbol": "ETHUSDT",
    "datetime": 1761197149286,
    "lable": -1, 
    "ma_vector":[0.0237,0.0176,0.0441],
    "price_short_vector":[-0.3671,1.077,-1.657,-1.351,-0.2234,0.6229,1.673,1.405,0.9997,-0.6702,-1.227,-0.7792,0.07501,-0.1578,0.5798],
    "price_long_vector":[-2.567,-2.174,-2.329,-1.973,-1.41,-1.169,-1.289,-1.237,-1.551,-1.145,-0.2718,-0.6041,-0.8523,-0.2534,-0.5139,-0.08981,-0.3628,-1.007,0.1119,0.5201,0.2245,0.1156,0.2145,1.046,1.087,0.8687,0.6902,0.8247,1.115,1.716,1.483,1.165,1.442,1.278,1.213,1.158,0.9653,-0.1224,0.1174,-0.7764,0.1387,0.4995,0.02676,-0.5028,0.276,0.06071,0.8457,-0.6403,-0.4741,0.1388,0.5989,1.17,1.024,0.8038,-0.1041,-0.4067,-0.1633,0.3011,0.1745,0.5755]
  }
]
```

## 功能列表

### 后端 (Go-Kratos gRPC backend + Qdrant vector DB)
-  **Proto API**: `api/vector/v1/vector.proto` - 完整的 gRPC 接口定义
  - Ingest -> 存储指标 + 向量
  - QuerySimilar(query, top_k, metrics=["cosine","manhattan"]) -> 最近邻搜索 (支持多相似度混合加权)
  - Update 更新向量
  - Delete -> 删除向量
  - List -> 分页列表
-  **向量嵌入**: `internal/biz/embedding.go` - 特征工程
  - 标准化处理 (close 归一化成向量)
-  **业务逻辑**: `internal/biz/vector.go` - 完整服务业务逻辑实现
-  **存储适配器**: 
  - `internal/data/vector_repo.go` - Qdrant向量数据库接口(增删改查，相似度匹配)
-  **HTTP网关**: 内置REST API网关，gRPC -> HTTP转换
-  **统一服务**: `cmd/klabel/main.go` - gRPC(:9000) + HTTP(:8080)
-  **结构体** `internal/model/vector.go` 定义实体对象
-  **工具方法** `internal/utils/*.go` 通用的工具方法

### 前端 (Frontend) 无需生成，忽略前端任务
-
## 🚀 快速启动

访问: http://localhost:8080

## 📋 接口验证

```powershell
# 新增向量
curl -X POST http://localhost:8080/api/vectors -H "Content-Type: application/json" -d '{
  "close": 112,
  "symbol": "ETHUSDT",
  "datetime": 1761197149286,
  "lable": -1, 
  "ma_vector":[0.0237,0.0176,0.0441],
  "price_short_vector":[-0.3671,1.077,-1.657,-1.351,-0.2234,0.6229,1.673,1.405,0.9997,-0.6702,-1.227,-0.7792,0.07501,-0.1578,0.5798],
  "price_long_vector":[-2.567,-2.174,-2.329,-1.973,-1.41,-1.169,-1.289,-1.237,-1.551,-1.145,-0.2718,-0.6041,-0.8523,-0.2534,-0.5139,-0.08981,-0.3628,-1.007,0.1119,0.5201,0.2245,0.1156,0.2145,1.046,1.087,0.8687,0.6902,0.8247,1.115,1.716,1.483,1.165,1.442,1.278,1.213,1.158,0.9653,-0.1224,0.1174,-0.7764,0.1387,0.4995,0.02676,-0.5028,0.276,0.06071,0.8457,-0.6403,-0.4741,0.1388,0.5989,1.17,1.024,0.8038,-0.1041,-0.4067,-0.1633,0.3011,0.1745,0.5755]
}'

# 查询列表
curl http://localhost:8080/api/vectors?limit=10

# 相似度搜索(price_long_vector实际长度是60，price_short_vector实际长度是15 )
curl -X POST http://localhost:8080/api/vectors/search -H "Content-Type: application/json" -d '{
  "symbol": "ETHUSDT",
  "vectors": [
    {
      "name": "ma_vector",
      "vector": [0.0237, 0.0176, 0.0441],
      "weight": 0.5,
      "metric": "manhattan"
    },
    {
      "name": "price_short_vector",
      "vector": [0.12, 0.34, 0.56],
      "weight": 0.5,
      "metric": "cosine"
    },
    {
      "name": "price_long_vector",
      "vector": [-0.3671, 1.077, -1.657],
      "weight": 0.8,
      "metric": "cosine"
    }
  ],
  "top_k": 10
}'
```

## 📁 项目结构

├── .gitignore
├── docker-compose.dev.yml # 本地开发用
├── docker-compose.yml # 线上部署用
├── Dockerfile
├── go.mod
├── LICENSE
├── Makefile
├── openapi.yaml
├── README.md # 项目启动说明
├── TODO.md  # 项目生成任务
├── api                        # 微服务 proto 文件及生成代码
│   └── klabel
│       └── v1
│           └── vector.proto
├── cmd                        # 项目启动入口
│   └── klabel
│       ├── main.go
│       ├── wire_gen.go
│       └── wire.go
├── configs                    # 配置文件
│   └── config.yaml
├── internal                   # 该服务所有不对外暴露的代码，通常的业务逻辑都在这下面，使用internal避免错误引用
│   ├── biz                    # 业务逻辑的组装层，类似 DDD 的 domain 层，data 类似 DDD 的 repo，而 repo 接口在这里定义，使用依赖倒置的原则。
│   │   ├── biz.go
│   │   └── vector.go
│   ├── conf                   # 内部使用的config的结构定义，使用proto格式生成
│   │   ├── conf.pb.go
│   │   ├── conf.proto
│   ├── data                   # 业务数据访问，包含 cache、db 等封装，实现了 biz 的 repo 接口。我们可能会把 data 与 dao 混淆在一起，data 偏重业务的含义，它所要做的是将领域对象重新拿出来，我们去掉了 DDD 的 infra层。
│   │   ├── data.go
│   │   └── vector_repo.go
│   ├── model                  # 定义实体对象
│   │   └── vector.go
│   ├── server                 # http和grpc实例的创建和配置
│   │   ├── grpc.go
│   │   ├── http.go
│   │   └── server.go
│   ├── service                # 实现了 api 定义的服务层，类似 DDD 的 application 层，处理 DTO 到 biz 领域实体的转换(DTO -> DO)，同时协同各类 biz 交互，但是不应处理复杂逻辑
│       └── vector.go
│   └── utils                  # 工具方法，如果有通用的方法抽离到此
├── test                       # 测试相关
│   └── rest-api.http
└── third_party                # 第三方 proto 依赖
    ├── README.md
    ├── errors
    │   └── errors.proto
    ├── google
    │   └── api
    │   └── protobuf
    ├── openapi
    │   └── v3
    │       ├── annotations.proto
    │       └── openapi.proto
    └── validate
        ├── README.md
        └── validate.proto