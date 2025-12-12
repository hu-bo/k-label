# Kline Vector Platform

## 一句话概览
后端通过  API 聚合交易数据、PostgreSQL）存储业务记录，Qdrant gRPC 通信，前端基于 umi + Ant Design Pro，整个工程可用 docker-compose 一键启动。

## 核心部署
- `docker-compose.dev.yml` 控制 Qdrant（含 UI）、PostgreSQL、Redis。
- `docker-compose.yml` 控制 Qdrant（含 UI）、server（Echo）、PostgreSQL、Redis。
- 命令：
  ```powershell
  cd e:\Project\my-project\k-label
  docker-compose up -d
  ```

## 首页关键模块
- Hero：平台介绍 + CTA
- K 线图：使用 klinecharts，返回数据时附带服务端缓存的 `tag_id`
- 策略列表：参考 OKX 交易机器人（仅展示，接口为 `GET /api/strategies`）
- 数据打标：列表、详情、更新接口（create, list, get, update）
- 向量模块：提供向量搜索与预计算（cosine / euclidean / pearson 或选择性启用）

## 主要接口
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/trader/v1/klines?symbo=BTC_USDT&exchange=1` | 拉取 K 线 + 指标，返回 `tag_id` 用于打标更新的缓存定位 |
| `GET` | `/trader/v1/strategies` | 策略列表（目前可 mock） |
| `POST` | `/klabel/v1/label/create` | 创建打标记录 (`symbol`, `type`, `metadata`) |
| `GET` | `/klabel/v1/label/list` | 打标列表（支持 `symbol`, `action`, `page` 等过滤） |
| `GET` | `/klabel/v1/label/{id}` | 打标详情，用于回显 `tag_id`/K 线区间 |
| `POST` | `/klabel/v1/label/update` | 处理 `action` 状态，`tag_id` 指向缓存的向量样本（服务端根据 `timestamp` 过滤） |
| `POST` | `/klabel/v1/vector-search` | 多向量搜索，`query`, `similarity_config`, `top_k`, `filters`（symbol、name、时间范围） |
| `POST` | `/klabel/v1/vector/precompute` | 预计算向量（可选方法 list），用于缓存/加速相似度计算 |

## 向量搜索示例
请求:
```json
{
  "query": {
    "15m_price_vec": [1,0.1,1,1,0.1,1],
    "4h_price_vec": [1,0.1,1,1,0.1,1],
    "1d_price_vec": [1,0.1,1,1,0.1,1]
  },
  "similarity_config": {
    "15m_price_vec": {"method":"cosine","weight":0.5,"normalize":true},
    "4h_price_vec": {"method":"euclidean","weight":0.3,"normalize":false},
    "1d_price_vec": {"method":"euclidean","weight":0.2,"normalize":true}
  },
  "top_k": 10,
  "filters": {"symbol":"BTC/USDT","name":"mid","start_time":1700000000000,"end_time":1710000000000}
}
```
响应示例:
```json
{
  "results": [
    {"id":"seg_8a3b2c","symbol":"BTC/USDT","name":"mid","timestamp":1705000000000,
     "similarity_score":0.923,
     "dimension_scores":{"15m_price_vec":0.95,"4h_price_vec":0.88,"1d_price_vec":0.85},
     "metadata":{"label":"buy"}}
  ],
  "total_matched":1240,
  "query_time_ms":42
}
```

## 打标更新说明
- `POST /api/label/update` 不传向量，传入 `symbol`, `timestamp`, `tag_id`。
- Node 层用 `tag_id` 找到 Redis/内存缓存的向量，按 `timestamp` 精确匹配后写入 PostgreSQL，并同步 Rust/Qdrant（若需要）。
- 响应示例:
```json
{
  "success": true,
  "message": "标签已更新",
  "data": {"id": "label_123456","action": 1}
}
```