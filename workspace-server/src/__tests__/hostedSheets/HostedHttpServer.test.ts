/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { describe, expect, it, jest } from '@jest/globals';
import { createHostedHttpServer } from '../../hostedSheets/httpServer';

describe('hosted Sheets HTTP boundary',()=>{
  it('rejects requests without exactly one syntactically valid Connection Key before MCP handling',async()=>{
    const service={readValues:jest.fn(),updateValues:jest.fn()} as any;
    const server=createHostedHttpServer(service);
    await new Promise<void>(resolve=>server.listen(0,'127.0.0.1',resolve));
    const address=server.address(); if(!address||typeof address==='string') throw new Error('missing address');
    try {
      const missing=await fetch(`http://127.0.0.1:${address.port}/mcp`,{method:'POST',body:'{}'});
      expect(missing.status).toBe(401);
      const invalid=await fetch(`http://127.0.0.1:${address.port}/mcp`,{method:'POST',headers:{'x-cuan-mcp-connection-key':'Bearer token'},body:'{}'});
      expect(invalid.status).toBe(401);
      expect(service.readValues).not.toHaveBeenCalled();
    } finally { await new Promise<void>((resolve,reject)=>server.close(error=>error?reject(error):resolve())); }
  });
  it('serves only the two hosted Sheets tools over stateless Streamable HTTP',async()=>{
    const service={readValues:jest.fn(),updateValues:jest.fn()} as any;
    const server=createHostedHttpServer(service);
    await new Promise<void>(resolve=>server.listen(0,'127.0.0.1',resolve));
    const address=server.address(); if(!address||typeof address==='string') throw new Error('missing address');
    const endpoint=`http://127.0.0.1:${address.port}/mcp`,key=`ci_mcp_ck_${'b'.repeat(64)}`;
    const headers={'content-type':'application/json',accept:'application/json, text/event-stream','x-cuan-mcp-connection-key':key};
    try {
      const initialized=await fetch(endpoint,{method:'POST',headers,body:JSON.stringify({jsonrpc:'2.0',id:1,method:'initialize',params:{protocolVersion:'2025-03-26',capabilities:{},clientInfo:{name:'test',version:'1'}}})});
      expect(initialized.status).toBe(200);
      const listed=await fetch(endpoint,{method:'POST',headers,body:JSON.stringify({jsonrpc:'2.0',id:2,method:'tools/list',params:{}})});
      expect(listed.status).toBe(200);
      const payload=await listed.json() as {result:{tools:{name:string}[]}};
      expect(payload.result.tools.map(tool=>tool.name).sort()).toEqual(['sheets_read_values','sheets_update_values']);
    } finally { await new Promise<void>((resolve,reject)=>server.close(error=>error?reject(error):resolve())); }
  });
});
