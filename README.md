# Kline Vector Platform

Enterprise-grade stack:  backend + Qdrant vector DB +  umi + ant-design-pro  frontend. Supports docker-compose one-command deploy.


Docker:

- A `docker-compose.yml` is provided at the repo root that starts Qdrant, Qdrant UI, and the `server` service.

To run everything locally:

```powershell
cd e:\Project\my-project\k-label
docker-compose up -d
```

umi + ant-design-pro: E:\Project\my-project\k-label\clinet
backend + Qdrant vector DB:  E:\Project\my-project\k-label\server

目标是构建一个精美、高性能、模块化的首页，包含以下核心功能：

首页介绍（Hero 区域）
K 线图展示（使用 klinecharts）
GET /api/v1/klines 返回k线图数据(服务端同时缓存对应的指标数据，通过随机tag_id缓存)

市场策略列表（参考 OKX 交易机器人页面）
GET /api/strategies (空接口逻辑暂时不用实现)

数据打标模块（打标列表 + 打标详情页）
POST /api/label/create (symbol, type)
GET /api/label/list
GET /api/label/{id}
POST /api/label/update (action=1心中action=2删除action=0未操作)


向量逻辑：
POST /api/v1/vector-search
POST /api/v1/vector/precompute   "methods": ["cosine", "euclidean", "pearson"]  // 需预计算的相似度方法(如果无法加速的的可以删除)

## 📋 接口示例

POST /api/v1/vector-search

{
  "query": {
    "15m_price_vec": [67200, 67350, 67100, 67400, 67500],
    "4h_price_vec": [67200, 67350, 67100, 67400, 67500],
    "1d_price_vec": [67200, 67350, 67100, 67400, 67500]
  },
  "similarity_config": {
    "15m_price_vec": {
      "method": "cosine",       // 可选: cosine, euclidean, pearson
      "weight": 0.5,
      "normalize": true
    },
    "4h_price_vec": {
      "method": "euclidean",
      "weight": 0.3,
      "normalize": false
    },
    "1d_price_vec": {
      "method": "euclidean",
      "weight": 0.2,
      "normalize": true
    }
  },
  "top_k": 10,
  "filters": {
    "symbol": "BTC/USDT",
    "name: "mid",
    "start_time": 1700000000000,
    "end_time": 1710000000000
  }
}

{
  "results": [
    {
      "id": "seg_8a3b2c",
      "symbol": "BTC/USDT",
      "name": "mid",
      "timestamp": 1705000000000,
      "similarity_score": 0.923,
      "dimension_scores": {
        "15m_price_vec": 0.95,
        "4h_price_vec": 0.88,
        "1d_price_vec": 0.85
      },
      "metadata": {
        "label": "buy",
      }
    },
    // ... 其他 top_k 条结果
  ],
  "total_matched": 1240,
  "query_time_ms": 42
}


POST /api/label/update
// 15m_price_vec、4h_price_vec、1d_price_vec数据量过大，不传给前端、也不是前端计算，
// 需要通过GET /api/v1/klines获取图表(服务端同时计算指标并缓存到内存中)，/api/label/update时，根据tag_id找到缓存，timestamp过滤查找到对应的vectors，写入数据库中
请求参数：
```
{
  "id": "label_123456",   // 标签ID
  "action": 1,             // action=1心中action=2删除action=0未操作
  "symbol": "BTC_USDT",
  "timestamp": 1705000000000,
  "tag_id": "0x8DE22F05F2D0F4A" // 命中缓存的id
}
```

返回示例：
```
{
  "success": true,
  "message": "标签已更新",
  "data": {
    "id": "label_123456",
    "action": 1
  }
}
```