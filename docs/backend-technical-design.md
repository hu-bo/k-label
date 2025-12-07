# K-Label 后端技术方案

## 一、技术架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        API Gateway                               │
│                    Midway.js (Koa)                               │
├─────────────────────────────────────────────────────────────────┤
│                      业务逻辑层                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ LabelService│  │KlineService │  │   VectorSearchService   │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                      数据访问层                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Prisma                                   ││
│  └─────────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────────┤
│                      数据存储层                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │           PostgreSQL + pgvector 扩展                        ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ ││
│  │  │ kline_data  │  │   labels    │  │  vector_segments    │ ││
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘ ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## 二、为什么选择 PostgreSQL + pgvector


### 2.2 pgvector 核心能力

- **支持向量类型**：`vector(n)` 存储 n 维浮点向量
- **距离函数**：
  - `<->` 欧几里得距离 (L2)
  - `<#>` 负内积 (用于 cosine 等)
  - `<=>` 余弦距离
- **索引类型**：
  - IVFFlat：适合中等规模数据
  - HNSW：适合大规模数据，查询更快

## 三、数据库设计

### 3.1 核心表结构

```sql
-- 启用 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;

-- K线数据段表
CREATE TABLE vector_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(32) NOT NULL,           -- 交易对 BTC/USDT
    period VARCHAR(16) NOT NULL,           -- 周期 1h, 4h, 1d
    timestamp BIGINT NOT NULL,             -- 时间戳

    -- 多维度向量存储
    price_vec vector(64),                  -- 价格向量 (支持最多64维)
    rsi_vec vector(64),                    -- RSI 向量
    volume_ratio_vec vector(64),           -- 成交量比率向量
    macd_vec vector(64),                   -- MACD 向量 (可选)

    -- 元数据
    label VARCHAR(32),                     -- 标签: buy, sell, hold
    confidence DECIMAL(5,4),               -- 置信度
    metadata JSONB DEFAULT '{}',           -- 扩展元数据

    -- 时间戳
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- 索引优化
    CONSTRAINT unique_segment UNIQUE (symbol, period, timestamp)
);

-- 创建向量索引 (HNSW - 推荐)
CREATE INDEX idx_price_vec ON vector_segments
    USING hnsw (price_vec vector_cosine_ops);

CREATE INDEX idx_rsi_vec ON vector_segments
    USING hnsw (rsi_vec vector_cosine_ops);

CREATE INDEX idx_volume_ratio_vec ON vector_segments
    USING hnsw (volume_ratio_vec vector_cosine_ops);

-- 创建过滤索引
CREATE INDEX idx_symbol_period ON vector_segments (symbol, period);
CREATE INDEX idx_timestamp ON vector_segments (timestamp);
CREATE INDEX idx_label ON vector_segments (label);

-- 标签表
CREATE TABLE labels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(32) NOT NULL,
    label_type VARCHAR(32) NOT NULL,       -- 标签类型
    start_time BIGINT NOT NULL,
    end_time BIGINT NOT NULL,
    description TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- K线原始数据表 (可选，用于回溯)
CREATE TABLE kline_data (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(32) NOT NULL,
    period VARCHAR(16) NOT NULL,
    timestamp BIGINT NOT NULL,
    open DECIMAL(20,8) NOT NULL,
    high DECIMAL(20,8) NOT NULL,
    low DECIMAL(20,8) NOT NULL,
    close DECIMAL(20,8) NOT NULL,
    volume DECIMAL(30,8) NOT NULL,
    CONSTRAINT unique_kline UNIQUE (symbol, period, timestamp)
);
```

### 3.2 向量维度设计

```typescript
// 向量维度配置
interface VectorDimensionConfig {
  price: {
    dimension: 32,      // 32个K线收盘价
    normalize: true,    // Z-score 标准化
  },
  rsi: {
    dimension: 14,      // 14个RSI值
    normalize: false,   // RSI已是0-100范围
  },
  volume_ratio: {
    dimension: 20,      // 20个成交量比率
    normalize: true,    // 标准化处理
  },
  macd: {
    dimension: 26,      // MACD序列
    normalize: true,
  }
}
```

## 四、混合权重向量查询实现

### 4.1 核心查询逻辑

```sql
-- 混合权重向量搜索 SQL
WITH similarity_scores AS (
    SELECT
        id,
        symbol,
        period,
        timestamp,
        label,
        metadata,
        -- 各维度相似度计算
        1 - (price_vec <=> $1::vector) AS price_sim,
        1 - (rsi_vec <=> $2::vector) AS rsi_sim,
        1 - (volume_ratio_vec <=> $3::vector) AS volume_sim
    FROM vector_segments
    WHERE
        symbol = $4
        AND period = $5
        AND timestamp BETWEEN $6 AND $7
)
SELECT
    *,
    -- 加权混合得分
    (price_sim * $8 + rsi_sim * $9 + volume_sim * $10) AS total_score
FROM similarity_scores
ORDER BY total_score DESC
LIMIT $11;
```

### 4.2 TypeScript 服务实现

```typescript
// src/service/vector.service.ts

import { Provide, Inject } from '@midwayjs/core';
import { InjectEntityManager } from '@midwayjs/typeorm';
import { EntityManager } from 'typeorm';

interface SimilarityConfig {
  method: 'cosine' | 'euclidean' | 'pearson';
  weight: number;
  normalize: boolean;
}

interface SearchParams {
  query: {
    price_vec?: number[];
    rsi_vec?: number[];
    volume_ratio_vec?: number[];
  };
  similarity_config: Record<string, SimilarityConfig>;
  top_k: number;
  filters: {
    symbol?: string;
    period?: string;
    start_time?: number;
    end_time?: number;
  };
}

@Provide()
export class VectorService {
  @InjectEntityManager()
  entityManager: EntityManager;

  /**
   * 混合权重向量搜索
   */
  async search(params: SearchParams) {
    const startTime = Date.now();
    const { query, similarity_config, top_k = 10, filters } = params;

    // 构建动态 SQL
    const selectClauses: string[] = [];
    const scoreComponents: string[] = [];
    const queryParams: any[] = [];
    let paramIndex = 1;

    // 处理每个向量维度
    const dimensions = ['price', 'rsi', 'volume_ratio'];

    for (const dim of dimensions) {
      const vec = query[`${dim}_vec`];
      const config = similarity_config[dim];

      if (vec && config) {
        const normalizedVec = config.normalize
          ? this.normalizeVector(vec, config.method)
          : vec;

        // 根据方法选择距离运算符
        const distanceOp = this.getDistanceOperator(config.method);

        selectClauses.push(
          `1 - (${dim}_vec ${distanceOp} $${paramIndex}::vector) AS ${dim}_sim`
        );
        queryParams.push(`[${normalizedVec.join(',')}]`);

        scoreComponents.push(`${dim}_sim * ${config.weight}`);
        paramIndex++;
      }
    }

    // 构建过滤条件
    const whereClauses: string[] = [];

    if (filters.symbol) {
      whereClauses.push(`symbol = $${paramIndex}`);
      queryParams.push(filters.symbol);
      paramIndex++;
    }

    if (filters.period) {
      whereClauses.push(`period = $${paramIndex}`);
      queryParams.push(filters.period);
      paramIndex++;
    }

    if (filters.start_time) {
      whereClauses.push(`timestamp >= $${paramIndex}`);
      queryParams.push(filters.start_time);
      paramIndex++;
    }

    if (filters.end_time) {
      whereClauses.push(`timestamp <= $${paramIndex}`);
      queryParams.push(filters.end_time);
      paramIndex++;
    }

    // 完整 SQL
    const sql = `
      WITH similarity_scores AS (
        SELECT
          id, symbol, period, timestamp, label, metadata,
          ${selectClauses.join(',\n          ')}
        FROM vector_segments
        ${whereClauses.length ? 'WHERE ' + whereClauses.join(' AND ') : ''}
      )
      SELECT
        *,
        (${scoreComponents.join(' + ')}) AS total_score
      FROM similarity_scores
      ORDER BY total_score DESC
      LIMIT $${paramIndex}
    `;

    queryParams.push(top_k);

    // 执行查询
    const results = await this.entityManager.query(sql, queryParams);

    // 格式化返回结果
    return {
      results: results.map((row: any) => ({
        id: row.id,
        symbol: row.symbol,
        period: row.period,
        timestamp: Number(row.timestamp),
        similarity_score: Number(row.total_score.toFixed(6)),
        dimension_scores: {
          price: row.price_sim ? Number(row.price_sim.toFixed(6)) : null,
          rsi: row.rsi_sim ? Number(row.rsi_sim.toFixed(6)) : null,
          volume_ratio: row.volume_sim ? Number(row.volume_sim.toFixed(6)) : null,
        },
        metadata: row.metadata || {},
      })),
      total_matched: results.length,
      query_time_ms: Date.now() - startTime,
    };
  }

  /**
   * 向量归一化
   */
  private normalizeVector(vec: number[], method: string): number[] {
    if (method === 'cosine') {
      // L2 归一化
      const norm = Math.sqrt(vec.reduce((sum, v) => sum + v * v, 0)) || 1;
      return vec.map(v => v / norm);
    }

    // Z-score 标准化
    const mean = vec.reduce((sum, v) => sum + v, 0) / vec.length;
    const std = Math.sqrt(
      vec.reduce((sum, v) => sum + Math.pow(v - mean, 2), 0) / vec.length
    ) || 1;
    return vec.map(v => (v - mean) / std);
  }

  /**
   * 获取 pgvector 距离运算符
   */
  private getDistanceOperator(method: string): string {
    switch (method) {
      case 'cosine':
        return '<=>'; // 余弦距离
      case 'euclidean':
        return '<->'; // L2 距离
      default:
        return '<=>'; // 默认余弦
    }
  }

  /**
   * Pearson 相关系数 (应用层计算)
   * pgvector 不原生支持，需要在应用层处理
   */
  private pearsonCorrelation(a: number[], b: number[]): number {
    const n = Math.min(a.length, b.length);
    if (n === 0) return 0;

    const meanA = a.slice(0, n).reduce((s, v) => s + v, 0) / n;
    const meanB = b.slice(0, n).reduce((s, v) => s + v, 0) / n;

    let num = 0, denA = 0, denB = 0;
    for (let i = 0; i < n; i++) {
      const da = a[i] - meanA;
      const db = b[i] - meanB;
      num += da * db;
      denA += da * da;
      denB += db * db;
    }

    const den = Math.sqrt(denA * denB);
    return den === 0 ? 0 : (num / den + 1) / 2;
  }

  /**
   * 添加向量数据
   */
  async addSegment(data: {
    symbol: string;
    period: string;
    timestamp: number;
    vectors: {
      price?: number[];
      rsi?: number[];
      volume_ratio?: number[];
    };
    label?: string;
    metadata?: Record<string, any>;
  }) {
    const sql = `
      INSERT INTO vector_segments
        (symbol, period, timestamp, price_vec, rsi_vec, volume_ratio_vec, label, metadata)
      VALUES
        ($1, $2, $3, $4::vector, $5::vector, $6::vector, $7, $8)
      ON CONFLICT (symbol, period, timestamp)
      DO UPDATE SET
        price_vec = EXCLUDED.price_vec,
        rsi_vec = EXCLUDED.rsi_vec,
        volume_ratio_vec = EXCLUDED.volume_ratio_vec,
        label = EXCLUDED.label,
        metadata = EXCLUDED.metadata,
        updated_at = NOW()
      RETURNING id
    `;

    const result = await this.entityManager.query(sql, [
      data.symbol,
      data.period,
      data.timestamp,
      data.vectors.price ? `[${data.vectors.price.join(',')}]` : null,
      data.vectors.rsi ? `[${data.vectors.rsi.join(',')}]` : null,
      data.vectors.volume_ratio ? `[${data.vectors.volume_ratio.join(',')}]` : null,
      data.label || null,
      JSON.stringify(data.metadata || {}),
    ]);

    return { id: result[0].id };
  }

  /**
   * 批量导入
   */
  async bulkImport(segments: any[]) {
    const batchSize = 1000;
    let imported = 0;

    for (let i = 0; i < segments.length; i += batchSize) {
      const batch = segments.slice(i, i + batchSize);

      // 使用 COPY 或批量 INSERT 提高性能
      const values = batch.map((seg, idx) => {
        const offset = idx * 8;
        return `($${offset + 1}, $${offset + 2}, $${offset + 3},
                 $${offset + 4}::vector, $${offset + 5}::vector,
                 $${offset + 6}::vector, $${offset + 7}, $${offset + 8})`;
      }).join(',');

      const params = batch.flatMap(seg => [
        seg.symbol,
        seg.period,
        seg.timestamp,
        seg.vectors?.price ? `[${seg.vectors.price.join(',')}]` : null,
        seg.vectors?.rsi ? `[${seg.vectors.rsi.join(',')}]` : null,
        seg.vectors?.volume_ratio ? `[${seg.vectors.volume_ratio.join(',')}]` : null,
        seg.label || null,
        JSON.stringify(seg.metadata || {}),
      ]);

      await this.entityManager.query(`
        INSERT INTO vector_segments
          (symbol, period, timestamp, price_vec, rsi_vec, volume_ratio_vec, label, metadata)
        VALUES ${values}
        ON CONFLICT (symbol, period, timestamp) DO NOTHING
      `, params);

      imported += batch.length;
    }

    return { imported };
  }
}
```

## 五、性能优化策略

### 5.1 索引优化

```sql
-- HNSW 索引参数调优
CREATE INDEX idx_price_vec_hnsw ON vector_segments
USING hnsw (price_vec vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- 查询时设置 ef_search (召回率与速度平衡)
SET hnsw.ef_search = 40;
```

### 5.2 分区表 (大数据量场景)

```sql
-- 按时间范围分区
CREATE TABLE vector_segments (
    -- 字段定义同上
) PARTITION BY RANGE (timestamp);

-- 创建分区
CREATE TABLE vector_segments_2024_q1
    PARTITION OF vector_segments
    FOR VALUES FROM (1704067200000) TO (1711929600000);

CREATE TABLE vector_segments_2024_q2
    PARTITION OF vector_segments
    FOR VALUES FROM (1711929600000) TO (1719792000000);
```

### 5.3 查询优化

```typescript
// 预热常用查询
async warmupCache() {
  const commonSymbols = ['BTC/USDT', 'ETH/USDT'];
  const periods = ['1h', '4h', '1d'];

  for (const symbol of commonSymbols) {
    for (const period of periods) {
      await this.entityManager.query(`
        SELECT count(*) FROM vector_segments
        WHERE symbol = $1 AND period = $2
      `, [symbol, period]);
    }
  }
}
```

## 六、Docker 部署配置

### 6.1 docker-compose.yml 更新

```yaml
version: '3.8'
services:
  postgres:
    image: pgvector/pgvector:pg16
    container_name: klabel-postgres
    restart: unless-stopped
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: klabel
      POSTGRES_PASSWORD: klabel_secure_password
      POSTGRES_DB: klabel
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./server/scripts/init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U klabel"]
      interval: 10s
      timeout: 5s
      retries: 5

  server:
    image: node:20-alpine
    container_name: klabel-server
    restart: unless-stopped
    working_dir: /data/app
    volumes:
      - ./server:/data/app
    ports:
      - "7001:7001"
    environment:
      DATABASE_URL: postgresql://klabel:klabel_secure_password@postgres:5432/klabel
      NODE_ENV: production
    command: sh -c "npm ci --silent && npm run start"
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  postgres_data:
```

### 6.2 数据库初始化脚本

```sql
-- server/scripts/init.sql
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- 用于文本模糊搜索

-- 创建表结构 (如上所述)
-- ...

-- 创建索引
-- ...

-- 插入测试数据 (可选)
INSERT INTO vector_segments (symbol, period, timestamp, price_vec, label)
VALUES
  ('BTC/USDT', '1h', 1700000000000, '[0.1,0.2,0.3,0.4,0.5]', 'buy'),
  ('BTC/USDT', '1h', 1700003600000, '[0.2,0.3,0.4,0.5,0.6]', 'hold');
```

## 七、API 接口设计

### 7.1 接口清单

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/vector-search | 混合权重向量搜索 |
| POST | /api/v1/klines | 添加K线数据 |
| POST | /api/v1/vector/bulk-import | 批量导入 |
| GET | /api/v1/vector/stats | 获取统计信息 |

### 7.2 请求/响应示例

```typescript
// POST /api/v1/vector-search
// Request
{
  "query": {
    "price_vec": [67200, 67350, 67100, 67400, 67500],
    "rsi_vec": [55.2, 58.7, 52.1, 60.3, 62.8],
    "volume_ratio_vec": [1.2, 1.5, 0.9, 2.1, 1.8]
  },
  "similarity_config": {
    "price": { "method": "cosine", "weight": 0.5, "normalize": true },
    "rsi": { "method": "euclidean", "weight": 0.3, "normalize": false },
    "volume_ratio": { "method": "euclidean", "weight": 0.2, "normalize": true }
  },
  "top_k": 10,
  "filters": {
    "symbol": "BTC/USDT",
    "period": "1h",
    "start_time": 1700000000000,
    "end_time": 1710000000000
  }
}

// Response
{
  "success": true,
  "results": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "symbol": "BTC/USDT",
      "period": "1h",
      "timestamp": 1705000000000,
      "similarity_score": 0.923,
      "dimension_scores": {
        "price": 0.95,
        "rsi": 0.88,
        "volume_ratio": 0.85
      },
      "metadata": { "label": "buy" }
    }
  ],
  "total_matched": 1240,
  "query_time_ms": 42
}
```

## 八、迁移计划

### 8.1 从内存方案迁移

1. **Phase 1**: 部署 PostgreSQL + pgvector
2. **Phase 2**: 实现新的 VectorService
3. **Phase 3**: 数据迁移脚本
4. **Phase 4**: 切换流量，验证
5. **Phase 5**: 移除旧的内存实现

### 8.2 兼容性保证

- API 接口保持不变
- 响应格式保持一致
- 支持平滑回滚

## 九、监控与运维

### 9.1 关键指标

- 查询延迟 P50/P95/P99
- 向量索引命中率
- 数据库连接池使用率
- 磁盘使用量

### 9.2 日志配置

```typescript
// 慢查询日志
if (queryTimeMs > 100) {
  logger.warn('Slow vector search', {
    query_time_ms: queryTimeMs,
    top_k: params.top_k,
    filters: params.filters,
  });
}
```

---

**文档版本**: v1.0
**最后更新**: 2024-12
**作者**: K-Label Team
