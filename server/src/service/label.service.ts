import { Provide } from '@midwayjs/core';

@Provide()
export class LabelService {
  private store: Record<string, any> = {};
  private seq = 1;

  async create(body: any) {
    const id = String(this.seq++);
    const item = { id, ...body };
    this.store[id] = item;
    return item;
  }

  async list() {
    return Object.values(this.store);
  }

  async get(id: string) {
    return this.store[id] || null;
  }

  async update(body: any) {
    const id = body.id;
    if (!id || !this.store[id]) throw new Error('Not found');
    this.store[id] = { ...this.store[id], ...body };
    return this.store[id];
  }
}
