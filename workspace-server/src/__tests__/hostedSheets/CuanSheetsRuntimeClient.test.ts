/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { describe, expect, it, jest } from '@jest/globals';
import { CuanSheetsRuntimeClient } from '../../hostedSheets/CuanSheetsRuntimeClient';

describe('CuanSheetsRuntimeClient', () => {
  it('forwards the connection key only to the private Cuan runtime with bounded action payload', async () => {
    const fetchFn=jest.fn(async (_url:RequestInfo|URL,_init?:RequestInit)=>new Response(JSON.stringify({ok:true}),{status:200}));
    const client=new CuanSheetsRuntimeClient('https://example.supabase.co/functions/v1/google-sheets-runtime','private-id','private-secret',fetchFn);
    const value=await client.invoke(`ci_mcp_ck_${'a'.repeat(64)}`,'resolveRead',{operation:'read_values'});
    expect(value).toEqual({ok:true});
    expect(fetchFn).toHaveBeenCalledTimes(1);
    expect(fetchFn.mock.calls[0][1]?.headers).toEqual(expect.objectContaining({
      'x-cuan-google-sheets-service-id':'private-id',
      'x-cuan-google-sheets-service-secret':'private-secret',
      'x-cuan-mcp-connection-key':`ci_mcp_ck_${'a'.repeat(64)}`,
    }));
  });
  it('rejects insecure runtime URLs at construction',()=>{
    expect(()=>new CuanSheetsRuntimeClient('http://example.test/runtime','id','secret')).toThrow();
  });
});
