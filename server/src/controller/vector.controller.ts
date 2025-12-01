import { Inject, Controller, Post, Body } from '@midwayjs/core';
import { Context } from '@midwayjs/koa';
import { VectorService } from '../service/vector.service';

@Controller('/api/v1')
export class VectorController {
  @Inject()
  ctx: Context;

  @Inject()
  vectorService: VectorService;

  @Post('/vector-search')
  async search(@Body() body: any) {
    const res = await this.vectorService.search(body);
    return { success: true, ...res };
  }

  @Post('/vector/precompute')
  async precompute(@Body() body: any) {
    const res = await this.vectorService.precompute(body || {});
    return { success: true, ...res };
  }

  @Post('/klines')
  async klines(@Body() body: any) {
    const res = await this.vectorService.addKlines(body);
    return { success: true, ...res };
  }
}
