/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { createHash, randomUUID } from 'node:crypto';
import type { CuanSheetsRuntime } from './CuanSheetsRuntimeClient';

type Cell = string | number | boolean;
type Grid = Cell[][];
type Fetch = typeof fetch;
type Inputs = { spreadsheetId: string; range: string; expectedOldValues?: Grid; newValues?: Grid; previewId?: string; executionId?: string; approvalDigest?: string; confirmed?: boolean };
export class HostedSheetsError extends Error { constructor(public code: string) { super(code); } }
const ID = /^[A-Za-z0-9_-]{20,100}$/;
const KEY = /^ci_mcp_ck_[0-9a-f]{64}$/;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const RANGE = /^'([A-Za-z0-9 _-]{1,60})'!([A-Z]{1,3})([1-9]\d{0,4}):([A-Z]{1,3})([1-9]\d{0,4})$/;
const scope = 'https://www.googleapis.com/auth/spreadsheets';
const column = (s: string) => [...s].reduce((n, c) => n * 26 + c.charCodeAt(0) - 64, 0);
const validCell = (v:unknown) => (typeof v==='string'&&v.length<=256&&![...v].some(ch=>ch.charCodeAt(0)<32&&![9,10,13].includes(ch.charCodeAt(0))) || typeof v==='number'&&Number.isFinite(v) || typeof v==='boolean');
function shape(id: string, range: string) {
  const m = RANGE.exec(range);
  if (!ID.test(id) || !m || range.length > 128) throw new HostedSheetsError('INVALID_TARGET');
  const height = Number(m[5]) - Number(m[3]) + 1, width = column(m[4]) - column(m[2]) + 1;
  if (height < 1 || width < 1 || height * width > 100) throw new HostedSheetsError('INVALID_TARGET');
  return { height, width };
}
function grid(value: unknown, dims: {height:number;width:number}, rectangular = false, protectFormulas = true): Grid {
  if (!Array.isArray(value) || value.length > dims.height || value.some(row => !Array.isArray(row) || row.length > dims.width || row.some((v:unknown) => !validCell(v)))) throw new HostedSheetsError('INVALID_VALUES');
  if (rectangular && (value.length !== dims.height || value.some(row => row.length !== dims.width))) throw new HostedSheetsError('INVALID_VALUES');
  const result = Array.from({length:dims.height}, (_,r) => Array.from({length:dims.width}, (_,c) => value[r]?.[c] ?? '')) as Grid;
  if (protectFormulas && result.some(row => row.some(v => typeof v === 'string' && /^\s*=/.test(v)))) throw new HostedSheetsError('FORMULA_PROTECTED');
  return result;
}
const digest = (v: unknown) => createHash('sha256').update(JSON.stringify(v)).digest('hex');

export class GoogleSheetsHostedService {
  constructor(private readonly deps: { runtime:CuanSheetsRuntime; fetchFn?:Fetch; now?:()=>number }) {}
  private checkKey(key:string) { if (!KEY.test(key)) throw new HostedSheetsError('UNAUTHENTICATED'); }
  private async credential(value:Record<string,unknown>, binding:{spreadsheetId:string;range:string;executionId:string;requestDigest:string}, claimed=false) {
    if (value.accessToken == null || value.spreadsheetId !== binding.spreadsheetId || value.allowedRange !== binding.range || value.executionId !== binding.executionId || value.requestDigest !== binding.requestDigest || value.scope !== scope || typeof value.expiresAt !== 'number' || value.expiresAt <= (this.deps.now ?? Date.now)() || (claimed && value.writeClaimed !== true)) throw new HostedSheetsError('CUAN_BINDING_MISMATCH');
    return String(value.accessToken);
  }
  private async call(token:string, id:string, range:string, method:'GET'|'PUT', values?:Grid) {
    const encoded = encodeURIComponent(range);
    const url = `https://sheets.googleapis.com/v4/spreadsheets/${id}/values/${encoded}?${method === 'GET' ? 'majorDimension=ROWS&valueRenderOption=FORMULA' : 'valueInputOption=RAW'}`;
    let response:Response;
    try { response = await (this.deps.fetchFn ?? fetch)(url, {method, redirect:'error', signal:AbortSignal.timeout(5000), headers:{authorization:`Bearer ${token}`,'content-type':'application/json'}, ...(method==='PUT'?{body:JSON.stringify({majorDimension:'ROWS',values})}:{})}); }
    catch { throw new HostedSheetsError('PROVIDER_OUTCOME_UNKNOWN'); }
    if (!response.ok) throw new HostedSheetsError(method === 'PUT' ? 'PROVIDER_OUTCOME_UNKNOWN' : 'PROVIDER_READ_FAILED');
    return response.json() as Promise<Record<string,unknown>>;
  }
  async readValues(key:string,input:Pick<Inputs,'spreadsheetId'|'range'>) {
    this.checkKey(key); const dims=shape(input.spreadsheetId,input.range), executionId=randomUUID(), requestDigest=digest(['read_values',input.spreadsheetId,input.range]);
    const token=await this.credential(await this.deps.runtime.invoke(key,'resolveRead',{operation:'read_values',...input,executionId,requestDigest}),{...input,executionId,requestDigest});
    const raw=await this.call(token,input.spreadsheetId,input.range,'GET');
    return {spreadsheetId:input.spreadsheetId,range:input.range,values:grid(raw.values ?? [],dims,false,false)};
  }
  async updateValues(key:string,input:Inputs):Promise<Record<string,unknown>> {
    this.checkKey(key); const dims=shape(input.spreadsheetId,input.range);
    const oldValues=grid(input.expectedOldValues,dims,true), newValues=grid(input.newValues,dims,true);
    if (!input.confirmed) {
      const executionId=randomUUID(), previewId=executionId, requestDigest=digest(['update_values',input.spreadsheetId,input.range,oldValues,newValues]);
      const token=await this.credential(await this.deps.runtime.invoke(key,'resolveRead',{operation:'preview_update',spreadsheetId:input.spreadsheetId,range:input.range,previewId,executionId,requestDigest,expectedOldValues:oldValues,newValues}),{spreadsheetId:input.spreadsheetId,range:input.range,executionId,requestDigest});
      const current=grid((await this.call(token,input.spreadsheetId,input.range,'GET')).values ?? [],dims,true);
      if (JSON.stringify(current)!==JSON.stringify(oldValues)) throw new HostedSheetsError('STALE_VALUES');
      return {spreadsheetId:input.spreadsheetId,range:input.range,oldValues,newValues,previewId,executionId,approvalDigest:requestDigest,requiresConfirmation:true};
    }
    if (!UUID.test(input.previewId ?? '') || !UUID.test(input.executionId ?? '') || input.previewId !== input.executionId || input.approvalDigest !== digest(['update_values',input.spreadsheetId,input.range,oldValues,newValues])) throw new HostedSheetsError('CONFIRMATION_REQUIRED');
    const requestDigest=input.approvalDigest, executionId=input.executionId!, binding={spreadsheetId:input.spreadsheetId,range:input.range,executionId,requestDigest};
    const auth=await this.credential(await this.deps.runtime.invoke(key,'resolveRead',{operation:'confirm_update',...binding,previewId:input.previewId,expectedOldValues:oldValues,newValues}),binding);
    const before=grid((await this.call(auth,input.spreadsheetId,input.range,'GET')).values ?? [],dims,true);
    if (JSON.stringify(before)!==JSON.stringify(oldValues)) throw new HostedSheetsError('STALE_VALUES');
    const token=await this.credential(await this.deps.runtime.invoke(key,'claimWrite',{operation:'update_values',...binding}),binding,true);
    let status:'CONFIRMED'|'UNKNOWN_OUTCOME'='UNKNOWN_OUTCOME';
    try {
      await this.call(token,input.spreadsheetId,input.range,'PUT',newValues);
      const after=grid((await this.call(token,input.spreadsheetId,input.range,'GET')).values ?? [],dims,true);
      if (JSON.stringify(after)!==JSON.stringify(newValues)) throw new Error('readback mismatch');
      status='CONFIRMED';
    } catch { status='UNKNOWN_OUTCOME'; }
    const ack=await this.deps.runtime.invoke(key,'finalizeWrite',{operation:'update_values',...binding,status});
    if (ack.finalized!==true || ack.operation!=='update_values' || ack.status!==status || ack.spreadsheetId!==binding.spreadsheetId || ack.range!==binding.range || ack.executionId!==binding.executionId || ack.requestDigest!==binding.requestDigest) throw new HostedSheetsError('CUAN_FINALIZE_MISMATCH');
    return {status,spreadsheetId:input.spreadsheetId,range:input.range,executionId:input.executionId};
  }
}
