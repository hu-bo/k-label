import { Provide } from '@midwayjs/core';

@Provide()
export class VectorService {
  private segments: Array<any> = [];

  async addKlines(payload: any) {
    const items = Array.isArray(payload) ? payload : [payload];
    for (const it of items) {
      this.segments.push({
        id: it.id || `seg_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
        symbol: it.symbol || it.metadata?.symbol || 'UNKNOWN',
        period: it.period || '1h',
        timestamp: it.timestamp || Date.now(),
        vectors: it.vectors || it.price_vec || {},
        metadata: it.metadata || {},
      });
    }
    return { added: items.length };
  }

  async precompute(body: { methods?: string[] }) {
    const methods = body?.methods || [];
    return { precomputed: methods, segments: this.segments.length };
  }

  private dot(a: number[], b: number[]) {
    return a.reduce((s, v, i) => s + v * (b[i] ?? 0), 0);
  }

  private norm(a: number[]) {
    return Math.sqrt(a.reduce((s, v) => s + v * v, 0));
  }

  private cosine(a: number[], b: number[]) {
    const na = this.norm(a);
    const nb = this.norm(b);
    if (na === 0 || nb === 0) return 0;
    return this.dot(a, b) / (na * nb);
  }

  private euclideanSimilarity(a: number[], b: number[]) {
    const dist = Math.sqrt(a.reduce((s, v, i) => s + Math.pow(v - (b[i] ?? 0), 2), 0));
    return 1 / (1 + dist);
  }

  private pearson(a: number[], b: number[]) {
    const n = Math.min(a.length, b.length);
    if (n === 0) return 0;
    const mean = (arr: number[]) => arr.slice(0, n).reduce((s, v) => s + v, 0) / n;
    const ma = mean(a);
    const mb = mean(b);
    let num = 0;
    let denA = 0;
    let denB = 0;
    for (let i = 0; i < n; i++) {
      const da = a[i] - ma;
      const db = b[i] - mb;
      num += da * db;
      denA += da * da;
      denB += db * db;
    }
    const den = Math.sqrt(denA * denB);
    if (den === 0) return 0;
    return (num / den + 1) / 2;
  }

  private maybeNormalize(vec: number[], method?: string, normalize?: boolean) {
    if (!normalize) return vec;
    if (method === 'cosine') {
      const n = this.norm(vec) || 1;
      return vec.map((v) => v / n);
    }
    const mean = vec.reduce((s, v) => s + v, 0) / (vec.length || 1);
    const std = Math.sqrt(vec.reduce((s, v) => s + Math.pow(v - mean, 2), 0) / (vec.length || 1)) || 1;
    return vec.map((v) => (v - mean) / std);
  }

  async search(body: any) {
    const { query = {}, similarity_config = {}, top_k = 10, filters = {} } = body || {};

    if (!this.segments.length) {
      return { results: [], total_matched: 0 };
    }

    const scores: Array<{ seg: any; score: number; dimension_scores: any }> = [];

    for (const seg of this.segments) {
      if (filters.symbol && seg.symbol !== filters.symbol) continue;
      if (filters.period && seg.period !== filters.period) continue;
      if (filters.start_time && seg.timestamp < filters.start_time) continue;
      if (filters.end_time && seg.timestamp > filters.end_time) continue;

      let totalScore = 0;
      const dimScores: any = {};
      for (const key of Object.keys(similarity_config || {})) {
        const conf = similarity_config[key] || {};
        const method = conf.method || 'cosine';
        const weight = typeof conf.weight === 'number' ? conf.weight : 1;
        const normalize = !!conf.normalize;

        const qvec = query[`${key}_vec`] || query[key] || [];
        const svec = seg.vectors?.[key] || seg.vectors?.[`${key}_vec`] || [];
        if (!qvec || !qvec.length || !svec || !svec.length) {
          dimScores[key] = 0;
          continue;
        }

        const qn = this.maybeNormalize(qvec, method, normalize);
        const sn = this.maybeNormalize(svec, method, normalize);
        let sc = 0;
        if (method === 'cosine') sc = this.cosine(qn, sn);
        else if (method === 'euclidean') sc = this.euclideanSimilarity(qn, sn);
        else if (method === 'pearson') sc = this.pearson(qn, sn);
        else sc = this.cosine(qn, sn);

        dimScores[key] = Number(sc.toFixed(6));
        totalScore += sc * weight;
      }

      scores.push({ seg, score: totalScore, dimension_scores: dimScores });
    }

    scores.sort((a, b) => b.score - a.score);

    const results = scores.slice(0, top_k).map((s) => ({
      id: s.seg.id,
      symbol: s.seg.symbol,
      period: s.seg.period,
      timestamp: s.seg.timestamp,
      similarity_score: Number(s.score.toFixed(6)),
      dimension_scores: s.dimension_scores,
      metadata: s.seg.metadata || {},
    }));

    return { results, total_matched: scores.length };
  }
}
