import { Inject, Controller, Get, Post, Body, Param } from '@midwayjs/core';
import { Context } from '@midwayjs/koa';
import { LabelService } from '../service/label.service';

@Controller('/api')
export class LabelController {
  @Inject()
  ctx: Context;

  @Inject()
  labelService: LabelService;

  @Post('/label/create')
  async create(@Body() body: any) {
    const item = await this.labelService.create(body);
    return { success: true, data: item };
  }

  @Get('/label/list')
  async list() {
    const items = await this.labelService.list();
    return { success: true, data: items };
  }

  @Get('/label/:id')
  async get(@Param('id') id: string) {
    const item = await this.labelService.get(id);
    return { success: !!item, data: item };
  }

  @Post('/label/update')
  async update(@Body() body: any) {
    try {
      const item = await this.labelService.update(body);
      return { success: true, data: item };
    } catch (err) {
      return { success: false, message: String(err.message || err) };
    }
  }

  @Get('/strategies')
  async strategies() {
    return { success: true, data: [] };
  }
}
