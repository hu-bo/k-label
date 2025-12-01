# Kline Vector Platform

Enterprise-grade stack: midwayjs backend + Qdrant vector DB +  umi + ant-design-pro  frontend. Supports docker-compose one-command deploy.


umi + ant-design-pro: E:\Project\my-project\k-label\clinet
midwayjs backend + Qdrant vector DB:  E:\Project\my-project\k-label\server

目标是构建一个精美、高性能、模块化的首页，包含以下核心功能：

首页介绍（Hero 区域）
K 线图展示（使用 klinecharts）
POST /api/v1/klines

市场策略列表（参考 OKX 交易机器人页面）
GET /api/strategies (空接口逻辑暂时不用实现)

数据打标模块（打标列表 + 打标详情页）
POST /api/label/create (symbol, type)
GET /api/label/list
GET /api/label/{id}
POST /api/label/update


向量逻辑：
POST /api/v1/vector-search
POST /api/v1/vector/precompute   "methods": ["cosine", "euclidean", "pearson"]  // 需预计算的相似度方法(如果无法加速的的可以删除)

## 📋 接口示例

POST /api/v1/vector-search

{
  "query": {
    "price_vec": [67200, 67350, 67100, 67400, 67500],
    "rsi_vec": [55.2, 58.7, 52.1, 60.3, 62.8],
    "volume_ratio_vec": [1.2, 1.5, 0.9, 2.1, 1.8]
  },
  "similarity_config": {
    "price": {
      "method": "cosine",       // 可选: cosine, euclidean, pearson
      "weight": 0.5,
      "normalize": true
    },
    "rsi": {
      "method": "euclidean",
      "weight": 0.3,
      "normalize": false
    },
    "volume_ratio": {
      "method": "euclidean",
      "weight": 0.2,
      "normalize": true
    }
  },
  "top_k": 10,
  "filters": {
    "symbol": "BTC/USDT",
    "period": "1h",
    "start_time": 1700000000000,
    "end_time": 1710000000000
  }
}

{
  "results": [
    {
      "id": "seg_8a3b2c",
      "symbol": "BTC/USDT",
      "period": "1h",
      "timestamp": 1705000000000,
      "similarity_score": 0.923,
      "dimension_scores": {
        "price": 0.95,
        "rsi": 0.88,
        "volume_ratio": 0.85
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