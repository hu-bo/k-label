// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/get_user */
export async function apicontrollerGetuser(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.apicontrollerGetuserParams,
  options?: { [key: string]: any }
) {
  return request<any>("/api/get_user", {
    method: "GET",
    params: {
      ...params,
      uid: undefined,
      ...params["uid"],
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/label/${param0} */
export async function labelcontrollerGet(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.labelcontrollerGetParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<any>(`/api/label/${param0}`, {
    method: "GET",
    params: { ...queryParams },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/label/create */
export async function labelcontrollerCreate(
  body: Record<string, any>,
  options?: { [key: string]: any }
) {
  return request<any>("/api/label/create", {
    method: "POST",
    headers: {
      "Content-Type": "text/plain",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/label/list */
export async function labelcontrollerList(options?: { [key: string]: any }) {
  return request<any>("/api/label/list", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/label/update */
export async function labelcontrollerUpdate(
  body: Record<string, any>,
  options?: { [key: string]: any }
) {
  return request<any>("/api/label/update", {
    method: "POST",
    headers: {
      "Content-Type": "text/plain",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/strategies */
export async function labelcontrollerStrategies(options?: {
  [key: string]: any;
}) {
  return request<any>("/api/strategies", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/klines */
export async function vectorcontrollerKlines(
  body: Record<string, any>,
  options?: { [key: string]: any }
) {
  return request<any>("/api/v1/klines", {
    method: "POST",
    headers: {
      "Content-Type": "text/plain",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/vector-search */
export async function vectorcontrollerSearch(
  body: Record<string, any>,
  options?: { [key: string]: any }
) {
  return request<any>("/api/v1/vector-search", {
    method: "POST",
    headers: {
      "Content-Type": "text/plain",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/vector/precompute */
export async function vectorcontrollerPrecompute(
  body: Record<string, any>,
  options?: { [key: string]: any }
) {
  return request<any>("/api/v1/vector/precompute", {
    method: "POST",
    headers: {
      "Content-Type": "text/plain",
    },
    data: body,
    ...(options || {}),
  });
}
