
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
