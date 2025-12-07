# K-Label 前端技术方案

## 一、技术架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        K-Label Frontend                          │
├─────────────────────────────────────────────────────────────────┤
│  UI Framework          │  Ant Design Pro + UmiJS                │
│  状态管理               │  @umijs/max (dva/model)                │
│  K线图表               │  KlineCharts                            │
│  HTTP Client           │  @umijs/max request (umi-request)       │
│  样式方案               │  CSS Modules + Ant Design Token        │
├─────────────────────────────────────────────────────────────────┤
│                        核心模块                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   首页模块   │  │ K线展示模块 │  │    向量搜索模块         │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  策略列表    │  │  数据打标   │  │    搜索结果可视化       │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## 二、项目结构

```
clinet/
├── config/
│   ├── config.ts              # UmiJS 配置
│   ├── routes.ts              # 路由配置
│   └── proxy.ts               # 开发代理
├── src/
│   ├── components/            # 公共组件
│   │   ├── KlineChart/        # K线图表组件
│   │   ├── VectorSearch/      # 向量搜索组件
│   │   ├── SimilarityConfig/  # 相似度配置器
│   │   └── ResultList/        # 搜索结果列表
│   ├── pages/
│   │   ├── Home/              # 首页
│   │   ├── Label/             # 数据打标模块
│   │   │   ├── List/          # 打标列表
│   │   │   └── Detail/        # 打标详情
│   │   ├── Strategy/          # 策略列表
│   │   └── Search/            # 向量搜索页
│   ├── services/              # API 服务
│   │   ├── kline.ts
│   │   ├── label.ts
│   │   ├── vector.ts
│   │   └── strategy.ts
│   ├── models/                # 状态管理
│   │   ├── kline.ts
│   │   ├── label.ts
│   │   └── vector.ts
│   ├── utils/                 # 工具函数
│   │   ├── vector.ts          # 向量计算工具
│   │   └── format.ts          # 格式化工具
│   └── app.tsx                # 应用入口
└── package.json
```

## 三、核心模块设计

### 3.1 K线图表模块

```typescript
// src/components/KlineChart/index.tsx
import { useEffect, useRef, useCallback } from 'react';
import { init, dispose, Chart } from 'klinecharts';

interface KlineChartProps {
  symbol: string;
  period: string;
  data?: KlineData[];
  onRangeSelect?: (start: number, end: number) => void;
  highlightRanges?: Array<{
    start: number;
    end: number;
    color: string;
    label?: string;
  }>;
}

interface KlineData {
  timestamp: number;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

const KlineChart: React.FC<KlineChartProps> = ({
  symbol,
  period,
  data,
  onRangeSelect,
  highlightRanges,
}) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<Chart | null>(null);

  useEffect(() => {
    if (!chartRef.current) return;

    // 初始化图表
    chartInstance.current = init(chartRef.current, {
      styles: {
        candle: {
          type: 'candle_solid',
          bar: {
            upColor: '#26A69A',
            downColor: '#EF5350',
            noChangeColor: '#888888',
          },
        },
        indicator: {
          lines: [
            { color: '#FF9800' },
            { color: '#2196F3' },
            { color: '#9C27B0' },
          ],
        },
      },
    });

    // 添加技术指标
    chartInstance.current.createIndicator('MA', false, { id: 'candle_pane' });
    chartInstance.current.createIndicator('VOL');
    chartInstance.current.createIndicator('RSI');

    return () => {
      if (chartInstance.current) {
        dispose(chartRef.current!);
      }
    };
  }, []);

  // 更新数据
  useEffect(() => {
    if (chartInstance.current && data?.length) {
      chartInstance.current.applyNewData(
        data.map((item) => ({
          timestamp: item.timestamp,
          open: item.open,
          high: item.high,
          low: item.low,
          close: item.close,
          volume: item.volume,
        }))
      );
    }
  }, [data]);

  // 高亮显示相似区间
  useEffect(() => {
    if (!chartInstance.current || !highlightRanges?.length) return;

    highlightRanges.forEach((range) => {
      chartInstance.current?.createOverlay({
        name: 'rect',
        points: [
          { timestamp: range.start, value: 0 },
          { timestamp: range.end, value: 0 },
        ],
        styles: {
          fill: range.color + '40', // 添加透明度
          stroke: range.color,
        },
        extendData: range.label,
      });
    });
  }, [highlightRanges]);

  // 框选功能
  const handleRangeSelect = useCallback(() => {
    if (!chartInstance.current || !onRangeSelect) return;

    chartInstance.current.subscribeAction('onCrosshairChange', (data) => {
      // 实现框选逻辑
    });
  }, [onRangeSelect]);

  return (
    <div
      ref={chartRef}
      style={{ width: '100%', height: '500px' }}
    />
  );
};

export default KlineChart;
```

### 3.2 向量搜索配置器

```typescript
// src/components/SimilarityConfig/index.tsx
import React from 'react';
import { Card, Form, Select, Slider, Switch, Space, Row, Col } from 'antd';

interface DimensionConfig {
  method: 'cosine' | 'euclidean' | 'pearson';
  weight: number;
  normalize: boolean;
}

interface SimilarityConfigProps {
  value?: Record<string, DimensionConfig>;
  onChange?: (value: Record<string, DimensionConfig>) => void;
  dimensions?: Array<{
    key: string;
    label: string;
    description?: string;
  }>;
}

const DEFAULT_DIMENSIONS = [
  { key: 'price', label: '价格向量', description: 'K线收盘价序列' },
  { key: 'rsi', label: 'RSI向量', description: '相对强弱指标' },
  { key: 'volume_ratio', label: '成交量比率', description: '相对成交量' },
];

const METHODS = [
  { value: 'cosine', label: '余弦相似度', description: '衡量方向相似性' },
  { value: 'euclidean', label: '欧氏距离', description: '衡量绝对距离' },
  { value: 'pearson', label: '皮尔逊相关', description: '衡量线性相关性' },
];

const SimilarityConfig: React.FC<SimilarityConfigProps> = ({
  value = {},
  onChange,
  dimensions = DEFAULT_DIMENSIONS,
}) => {
  const handleChange = (key: string, field: string, fieldValue: any) => {
    const newValue = {
      ...value,
      [key]: {
        method: 'cosine' as const,
        weight: 0.33,
        normalize: true,
        ...value[key],
        [field]: fieldValue,
      },
    };
    onChange?.(newValue);
  };

  // 计算权重总和
  const totalWeight = Object.values(value).reduce(
    (sum, config) => sum + (config?.weight || 0),
    0
  );

  return (
    <Card
      title="相似度配置"
      extra={
        <span style={{ color: totalWeight !== 1 ? '#ff4d4f' : '#52c41a' }}>
          权重总和: {totalWeight.toFixed(2)}
        </span>
      }
    >
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        {dimensions.map((dim) => (
          <Card
            key={dim.key}
            size="small"
            title={dim.label}
            extra={dim.description}
          >
            <Row gutter={16}>
              <Col span={8}>
                <Form.Item label="计算方法">
                  <Select
                    value={value[dim.key]?.method || 'cosine'}
                    onChange={(v) => handleChange(dim.key, 'method', v)}
                    options={METHODS}
                  />
                </Form.Item>
              </Col>
              <Col span={10}>
                <Form.Item label={`权重: ${(value[dim.key]?.weight || 0.33).toFixed(2)}`}>
                  <Slider
                    min={0}
                    max={1}
                    step={0.05}
                    value={value[dim.key]?.weight || 0.33}
                    onChange={(v) => handleChange(dim.key, 'weight', v)}
                  />
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item label="归一化">
                  <Switch
                    checked={value[dim.key]?.normalize ?? true}
                    onChange={(v) => handleChange(dim.key, 'normalize', v)}
                  />
                </Form.Item>
              </Col>
            </Row>
          </Card>
        ))}
      </Space>
    </Card>
  );
};

export default SimilarityConfig;
```

### 3.3 搜索结果列表

```typescript
// src/components/ResultList/index.tsx
import React from 'react';
import { List, Card, Tag, Progress, Space, Typography, Tooltip } from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined, MinusOutlined } from '@ant-design/icons';

const { Text } = Typography;

interface SearchResult {
  id: string;
  symbol: string;
  period: string;
  timestamp: number;
  similarity_score: number;
  dimension_scores: {
    price?: number;
    rsi?: number;
    volume_ratio?: number;
  };
  metadata: {
    label?: string;
  };
}

interface ResultListProps {
  results: SearchResult[];
  loading?: boolean;
  onItemClick?: (item: SearchResult) => void;
  onItemHover?: (item: SearchResult | null) => void;
}

const LABEL_COLORS: Record<string, string> = {
  buy: 'green',
  sell: 'red',
  hold: 'blue',
};

const LABEL_ICONS: Record<string, React.ReactNode> = {
  buy: <ArrowUpOutlined />,
  sell: <ArrowDownOutlined />,
  hold: <MinusOutlined />,
};

const ResultList: React.FC<ResultListProps> = ({
  results,
  loading,
  onItemClick,
  onItemHover,
}) => {
  const formatTimestamp = (ts: number) => {
    return new Date(ts).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getScoreColor = (score: number) => {
    if (score >= 0.9) return '#52c41a';
    if (score >= 0.7) return '#1890ff';
    if (score >= 0.5) return '#faad14';
    return '#ff4d4f';
  };

  return (
    <List
      loading={loading}
      dataSource={results}
      renderItem={(item, index) => (
        <List.Item
          onClick={() => onItemClick?.(item)}
          onMouseEnter={() => onItemHover?.(item)}
          onMouseLeave={() => onItemHover?.(null)}
          style={{ cursor: 'pointer' }}
        >
          <Card
            size="small"
            style={{ width: '100%' }}
            hoverable
          >
            <Space direction="vertical" style={{ width: '100%' }}>
              {/* 头部信息 */}
              <Space style={{ justifyContent: 'space-between', width: '100%' }}>
                <Space>
                  <Text strong>#{index + 1}</Text>
                  <Text>{item.symbol}</Text>
                  <Tag>{item.period}</Tag>
                  {item.metadata.label && (
                    <Tag
                      color={LABEL_COLORS[item.metadata.label]}
                      icon={LABEL_ICONS[item.metadata.label]}
                    >
                      {item.metadata.label.toUpperCase()}
                    </Tag>
                  )}
                </Space>
                <Text type="secondary">{formatTimestamp(item.timestamp)}</Text>
              </Space>

              {/* 总体相似度 */}
              <div>
                <Text>总体相似度: </Text>
                <Text
                  strong
                  style={{ color: getScoreColor(item.similarity_score), fontSize: 18 }}
                >
                  {(item.similarity_score * 100).toFixed(1)}%
                </Text>
              </div>

              {/* 各维度得分 */}
              <Space style={{ width: '100%' }} wrap>
                {item.dimension_scores.price !== undefined && (
                  <Tooltip title="价格相似度">
                    <div style={{ width: 120 }}>
                      <Text type="secondary" style={{ fontSize: 12 }}>价格</Text>
                      <Progress
                        percent={item.dimension_scores.price * 100}
                        size="small"
                        strokeColor={getScoreColor(item.dimension_scores.price)}
                        format={(p) => `${p?.toFixed(0)}%`}
                      />
                    </div>
                  </Tooltip>
                )}
                {item.dimension_scores.rsi !== undefined && (
                  <Tooltip title="RSI相似度">
                    <div style={{ width: 120 }}>
                      <Text type="secondary" style={{ fontSize: 12 }}>RSI</Text>
                      <Progress
                        percent={item.dimension_scores.rsi * 100}
                        size="small"
                        strokeColor={getScoreColor(item.dimension_scores.rsi)}
                        format={(p) => `${p?.toFixed(0)}%`}
                      />
                    </div>
                  </Tooltip>
                )}
                {item.dimension_scores.volume_ratio !== undefined && (
                  <Tooltip title="成交量相似度">
                    <div style={{ width: 120 }}>
                      <Text type="secondary" style={{ fontSize: 12 }}>成交量</Text>
                      <Progress
                        percent={item.dimension_scores.volume_ratio * 100}
                        size="small"
                        strokeColor={getScoreColor(item.dimension_scores.volume_ratio)}
                        format={(p) => `${p?.toFixed(0)}%`}
                      />
                    </div>
                  </Tooltip>
                )}
              </Space>
            </Space>
          </Card>
        </List.Item>
      )}
    />
  );
};

export default ResultList;
```

## 四、向量搜索页面

```typescript
// src/pages/Search/index.tsx
import React, { useState, useCallback } from 'react';
import { PageContainer, ProCard } from '@ant-design/pro-components';
import { Row, Col, Form, Select, Button, DatePicker, InputNumber, message, Spin } from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import { useRequest } from '@umijs/max';
import dayjs from 'dayjs';

import KlineChart from '@/components/KlineChart';
import SimilarityConfig from '@/components/SimilarityConfig';
import ResultList from '@/components/ResultList';
import { vectorSearch, getKlines } from '@/services/vector';

const { RangePicker } = DatePicker;

const SYMBOLS = [
  { value: 'BTC/USDT', label: 'BTC/USDT' },
  { value: 'ETH/USDT', label: 'ETH/USDT' },
  { value: 'SOL/USDT', label: 'SOL/USDT' },
];

const PERIODS = [
  { value: '15m', label: '15分钟' },
  { value: '4h', label: '4小时' },
  { value: '1d', label: '1天' },
];

const SearchPage: React.FC = () => {
  const [form] = Form.useForm();
  const [selectedRange, setSelectedRange] = useState<{ start: number; end: number } | null>(null);
  const [queryVectors, setQueryVectors] = useState<Record<string, number[]>>({});
  const [highlightRanges, setHighlightRanges] = useState<any[]>([]);

  // 加载 K 线数据
  const {
    data: klineData,
    loading: klineLoading,
    run: loadKlines,
  } = useRequest(
    async () => {
      const { symbol, period, timeRange } = form.getFieldsValue();
      if (!symbol || !period) return [];

      const params: any = { symbol, period };
      if (timeRange) {
        params.start_time = timeRange[0].valueOf();
        params.end_time = timeRange[1].valueOf();
      }

      const res = await getKlines(params);
      return res.data || [];
    },
    { manual: true }
  );

  // 向量搜索
  const {
    data: searchResults,
    loading: searchLoading,
    run: doSearch,
  } = useRequest(
    async () => {
      const values = form.getFieldsValue();

      if (Object.keys(queryVectors).length === 0) {
        message.warning('请先在K线图上框选查询区间');
        return { results: [], total_matched: 0 };
      }

      const params = {
        query: queryVectors,
        similarity_config: values.similarity_config || {
          price: { method: 'cosine', weight: 0.5, normalize: true },
          rsi: { method: 'euclidean', weight: 0.3, normalize: false },
          volume_ratio: { method: 'euclidean', weight: 0.2, normalize: true },
        },
        top_k: values.top_k || 10,
        filters: {
          symbol: values.symbol,
          period: values.period,
          start_time: values.timeRange?.[0]?.valueOf(),
          end_time: values.timeRange?.[1]?.valueOf(),
        },
      };

      const res = await vectorSearch(params);
      return res;
    },
    { manual: true }
  );

  // 处理框选
  const handleRangeSelect = useCallback(
    (start: number, end: number) => {
      setSelectedRange({ start, end });

      // 从 K 线数据中提取向量
      if (klineData) {
        const rangeData = klineData.filter(
          (k: any) => k.timestamp >= start && k.timestamp <= end
        );

        if (rangeData.length > 0) {
          setQueryVectors({
            price_vec: rangeData.map((k: any) => k.close),
            // RSI 和 volume_ratio 需要额外计算
          });
          message.success(`已选择 ${rangeData.length} 根K线`);
        }
      }
    },
    [klineData]
  );

  // 处理搜索结果高亮
  const handleResultHover = useCallback((item: any | null) => {
    if (!item) {
      setHighlightRanges([]);
      return;
    }

    setHighlightRanges([
      {
        start: item.timestamp,
        end: item.timestamp + 3600000 * 5, // 假设5个周期
        color: '#1890ff',
        label: `相似度: ${(item.similarity_score * 100).toFixed(1)}%`,
      },
    ]);
  }, []);

  return (
    <PageContainer
      title="向量相似搜索"
      subTitle="基于混合权重的K线形态搜索"
    >
      <Row gutter={[16, 16]}>
        {/* 左侧: K线图表 */}
        <Col xs={24} lg={16}>
          <ProCard title="K线图表" loading={klineLoading}>
            <Form form={form} layout="inline" style={{ marginBottom: 16 }}>
              <Form.Item name="symbol" label="交易对" initialValue="BTC/USDT">
                <Select options={SYMBOLS} style={{ width: 140 }} />
              </Form.Item>
              <Form.Item name="period" label="周期" initialValue="1h">
                <Select options={PERIODS} style={{ width: 100 }} />
              </Form.Item>
              <Form.Item name="timeRange" label="时间范围">
                <RangePicker
                  showTime
                  presets={[
                    { label: '最近7天', value: [dayjs().subtract(7, 'd'), dayjs()] },
                    { label: '最近30天', value: [dayjs().subtract(30, 'd'), dayjs()] },
                    { label: '最近90天', value: [dayjs().subtract(90, 'd'), dayjs()] },
                  ]}
                />
              </Form.Item>
              <Form.Item>
                <Button icon={<ReloadOutlined />} onClick={() => loadKlines()}>
                  加载数据
                </Button>
              </Form.Item>
            </Form>

            <KlineChart
              symbol={form.getFieldValue('symbol')}
              period={form.getFieldValue('period')}
              data={klineData}
              onRangeSelect={handleRangeSelect}
              highlightRanges={highlightRanges}
            />

            {selectedRange && (
              <div style={{ marginTop: 8, color: '#1890ff' }}>
                已选区间: {new Date(selectedRange.start).toLocaleString()} -{' '}
                {new Date(selectedRange.end).toLocaleString()}
              </div>
            )}
          </ProCard>
        </Col>

        {/* 右侧: 搜索配置 */}
        <Col xs={24} lg={8}>
          <ProCard title="搜索配置">
            <Form form={form} layout="vertical">
              <Form.Item name="similarity_config">
                <SimilarityConfig />
              </Form.Item>

              <Form.Item name="top_k" label="返回数量" initialValue={10}>
                <InputNumber min={1} max={100} style={{ width: '100%' }} />
              </Form.Item>

              <Form.Item>
                <Button
                  type="primary"
                  icon={<SearchOutlined />}
                  onClick={() => doSearch()}
                  loading={searchLoading}
                  block
                  size="large"
                >
                  搜索相似形态
                </Button>
              </Form.Item>
            </Form>
          </ProCard>
        </Col>

        {/* 搜索结果 */}
        <Col span={24}>
          <ProCard
            title="搜索结果"
            extra={
              searchResults && (
                <span>
                  共匹配 {searchResults.total_matched} 条，
                  耗时 {searchResults.query_time_ms}ms
                </span>
              )
            }
          >
            <Spin spinning={searchLoading}>
              <ResultList
                results={searchResults?.results || []}
                onItemHover={handleResultHover}
                onItemClick={(item) => {
                  // 跳转到详情或高亮显示
                  message.info(`查看详情: ${item.id}`);
                }}
              />
            </Spin>
          </ProCard>
        </Col>
      </Row>
    </PageContainer>
  );
};

export default SearchPage;
```

## 五、API 服务层

```typescript
// src/services/vector.ts
import { request } from '@umijs/max';

export interface VectorSearchParams {
  query: {
    price_vec?: number[];
    rsi_vec?: number[];
    volume_ratio_vec?: number[];
  };
  similarity_config: Record<
    string,
    {
      method: 'cosine' | 'euclidean' | 'pearson';
      weight: number;
      normalize: boolean;
    }
  >;
  top_k: number;
  filters: {
    symbol?: string;
    period?: string;
    start_time?: number;
    end_time?: number;
  };
}

export interface VectorSearchResult {
  success: boolean;
  results: Array<{
    id: string;
    symbol: string;
    period: string;
    timestamp: number;
    similarity_score: number;
    dimension_scores: {
      price?: number;
      rsi?: number;
      volume_ratio?: number;
    };
    metadata: Record<string, any>;
  }>;
  total_matched: number;
  query_time_ms: number;
}

/**
 * 向量相似搜索
 */
export async function vectorSearch(params: VectorSearchParams): Promise<VectorSearchResult> {
  return request('/api/v1/vector-search', {
    method: 'POST',
    data: params,
  });
}

/**
 * 获取K线数据
 */
export async function getKlines(params: {
  symbol: string;
  period: string;
  start_time?: number;
  end_time?: number;
}) {
  return request('/api/v1/klines', {
    method: 'POST',
    data: params,
  });
}

/**
 * 预计算向量
 */
export async function precompute(methods: string[]) {
  return request('/api/v1/vector/precompute', {
    method: 'POST',
    data: { methods },
  });
}
```

## 六、状态管理

```typescript
// src/models/vector.ts
import { useState, useCallback } from 'react';

interface VectorState {
  queryVectors: Record<string, number[]>;
  searchResults: any[];
  selectedResult: any | null;
  isSearching: boolean;
}

export default function useVectorModel() {
  const [queryVectors, setQueryVectors] = useState<Record<string, number[]>>({});
  const [searchResults, setSearchResults] = useState<any[]>([]);
  const [selectedResult, setSelectedResult] = useState<any | null>(null);
  const [isSearching, setIsSearching] = useState(false);

  // 从K线数据提取向量
  const extractVectors = useCallback((klines: any[], windowSize: number = 32) => {
    if (klines.length < windowSize) return;

    const priceVec = klines.slice(-windowSize).map((k) => k.close);

    // 计算 RSI
    const rsiVec = calculateRSI(klines, 14).slice(-14);

    // 计算成交量比率
    const volumeRatioVec = calculateVolumeRatio(klines, 20).slice(-20);

    setQueryVectors({
      price_vec: priceVec,
      rsi_vec: rsiVec,
      volume_ratio_vec: volumeRatioVec,
    });
  }, []);

  return {
    queryVectors,
    searchResults,
    selectedResult,
    isSearching,
    setQueryVectors,
    setSearchResults,
    setSelectedResult,
    setIsSearching,
    extractVectors,
  };
}

// 辅助函数: 计算 RSI
function calculateRSI(klines: any[], period: number = 14): number[] {
  const closes = klines.map((k) => k.close);
  const rsi: number[] = [];

  for (let i = period; i < closes.length; i++) {
    let gains = 0;
    let losses = 0;

    for (let j = i - period + 1; j <= i; j++) {
      const change = closes[j] - closes[j - 1];
      if (change > 0) gains += change;
      else losses -= change;
    }

    const avgGain = gains / period;
    const avgLoss = losses / period;
    const rs = avgLoss === 0 ? 100 : avgGain / avgLoss;
    rsi.push(100 - 100 / (1 + rs));
  }

  return rsi;
}

// 辅助函数: 计算成交量比率
function calculateVolumeRatio(klines: any[], period: number = 20): number[] {
  const volumes = klines.map((k) => k.volume);
  const ratios: number[] = [];

  for (let i = period; i < volumes.length; i++) {
    const avgVolume =
      volumes.slice(i - period, i).reduce((a, b) => a + b, 0) / period;
    ratios.push(avgVolume === 0 ? 1 : volumes[i] / avgVolume);
  }

  return ratios;
}
```

## 七、工具函数

```typescript
// src/utils/vector.ts

/**
 * 余弦相似度
 */
export function cosineSimilarity(a: number[], b: number[]): number {
  const dotProduct = a.reduce((sum, val, i) => sum + val * (b[i] || 0), 0);
  const normA = Math.sqrt(a.reduce((sum, val) => sum + val * val, 0));
  const normB = Math.sqrt(b.reduce((sum, val) => sum + val * val, 0));

  if (normA === 0 || normB === 0) return 0;
  return dotProduct / (normA * normB);
}

/**
 * 欧氏距离转相似度
 */
export function euclideanSimilarity(a: number[], b: number[]): number {
  const distance = Math.sqrt(
    a.reduce((sum, val, i) => sum + Math.pow(val - (b[i] || 0), 2), 0)
  );
  return 1 / (1 + distance);
}

/**
 * 皮尔逊相关系数
 */
export function pearsonCorrelation(a: number[], b: number[]): number {
  const n = Math.min(a.length, b.length);
  if (n === 0) return 0;

  const meanA = a.slice(0, n).reduce((s, v) => s + v, 0) / n;
  const meanB = b.slice(0, n).reduce((s, v) => s + v, 0) / n;

  let num = 0;
  let denA = 0;
  let denB = 0;

  for (let i = 0; i < n; i++) {
    const da = a[i] - meanA;
    const db = b[i] - meanB;
    num += da * db;
    denA += da * da;
    denB += db * db;
  }

  const den = Math.sqrt(denA * denB);
  return den === 0 ? 0 : (num / den + 1) / 2; // 归一化到 [0, 1]
}

/**
 * Z-score 标准化
 */
export function zScoreNormalize(vec: number[]): number[] {
  const mean = vec.reduce((s, v) => s + v, 0) / vec.length;
  const std =
    Math.sqrt(vec.reduce((s, v) => s + Math.pow(v - mean, 2), 0) / vec.length) || 1;
  return vec.map((v) => (v - mean) / std);
}

/**
 * L2 归一化
 */
export function l2Normalize(vec: number[]): number[] {
  const norm = Math.sqrt(vec.reduce((s, v) => s + v * v, 0)) || 1;
  return vec.map((v) => v / norm);
}
```

## 八、路由配置

```typescript
// config/routes.ts
export default [
  {
    path: '/',
    redirect: '/home',
  },
  {
    path: '/home',
    name: '首页',
    icon: 'home',
    component: './Home',
  },
  {
    path: '/search',
    name: '向量搜索',
    icon: 'search',
    component: './Search',
  },
  {
    path: '/label',
    name: '数据打标',
    icon: 'tags',
    routes: [
      {
        path: '/label/list',
        name: '打标列表',
        component: './Label/List',
      },
      {
        path: '/label/:id',
        name: '打标详情',
        component: './Label/Detail',
        hideInMenu: true,
      },
    ],
  },
  {
    path: '/strategy',
    name: '策略列表',
    icon: 'robot',
    component: './Strategy',
  },
];
```

## 九、代理配置

```typescript
// config/proxy.ts
export default {
  dev: {
    '/api/': {
      target: 'http://localhost:7001',
      changeOrigin: true,
    },
  },
  test: {
    '/api/': {
      target: 'http://test-server:7001',
      changeOrigin: true,
    },
  },
  pre: {
    '/api/': {
      target: 'http://pre-server:7001',
      changeOrigin: true,
    },
  },
};
```

## 十、依赖清单(模板已生成/clinet)

## 十一、性能优化

### 11.1 K线图表优化

- 使用虚拟滚动加载大量数据
- 防抖处理用户交互
- 按需加载技术指标

### 11.2 搜索结果优化

- 分页加载
- 结果缓存
- 骨架屏加载

### 11.3 状态管理优化

- 使用 `useMemo` 缓存计算结果
- 避免不必要的重渲染
- 按模块拆分状态

---

**文档版本**: v1.0
**最后更新**: 2024-12
**作者**: K-Label Team
