# K-Label Server API Spec

This file documents the basic API surfaces implemented for quick local development and testing.

## POST /api/v1/vector-search

Request JSON:

{
  "query": { /* optional: price_vec, rsi_vec, volume_ratio_vec */ },
  "similarity_config": {
    "price": { "method": "cosine|euclidean|pearson", "weight": 0.5, "normalize": true },
    "rsi": { "method": "euclidean", "weight": 0.3 },
    "volume_ratio": { "method": "euclidean", "weight": 0.2 }
  },
  "top_k": 10,
  "filters": { "symbol": "BTC/USDT", "period": "1h", "start_time": 0, "end_time": 0 }
}

Response:

{
  "success": true,
  "results": [ { "id", "symbol", "period", "timestamp", "similarity_score", "dimension_scores", "metadata" } ],
  "total_matched": 0
}

## POST /api/v1/vector/precompute

Request JSON: { "methods": ["cosine", "euclidean", "pearson"] }

Response: { "success": true, "precomputed": [...], "segments": <number> }

## POST /api/v1/klines

Accepts single kline object or array. Minimal shape accepted: { id?, symbol, period?, timestamp?, vectors?, metadata? }

Response: { "success": true, "added": <n> }

## Label APIs

- POST /api/label/create -> body: { symbol, type, ... } -> { success: true, data: { id, ... } }
- GET /api/label/list -> { success: true, data: [...] }
- GET /api/label/{id} -> { success: true, data: {...} }
- POST /api/label/update -> body must include `id` -> { success: true, data: {...} }

## GET /api/strategies

Returns placeholder empty array: { success: true, data: [] }

---
Notes:
- These implementations are in-memory stubs for local development and tests. Replace with persistent storage and a proper vector DB (e.g., Qdrant) in production.
