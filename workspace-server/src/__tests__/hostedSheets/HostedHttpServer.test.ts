/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { describe, expect, it, jest } from '@jest/globals';
import { request as httpRequest } from 'node:http';
import { createHostedHttpServer } from '../../hostedSheets/httpServer';

const allowedHost='sheets-mcp.cuaninsight.com';
const ingressSecret='i'.repeat(40);
function createServer(service:any) { return createHostedHttpServer(service,{allowedHost,ingressSecret}); }
function post(endpoint:string,headers:Record<string,string>,body:string) {
  return new Promise<{status:number;json:()=>Promise<any>}>((resolve,reject)=>{
    const request=httpRequest(endpoint,{method:'POST',headers},response=>{
      const chunks:Buffer[]=[];response.on('data',chunk=>chunks.push(Buffer.from(chunk)));
      response.on('end',()=>resolve({status:response.statusCode??0,json:async()=>JSON.parse(Buffer.concat(chunks).toString('utf8'))}));
    });
    request.on('error',reject);request.end(body);
  });
}

describe('hosted Sheets HTTP boundary',()=>{
  it('refuses startup with a missing Host binding or short ingress secret',()=>{
    const service={readValues:jest.fn(),updateValues:jest.fn()} as any;
    expect(()=>createHostedHttpServer(service,{allowedHost,ingressSecret:'short'})).toThrow('CUAN_SHEETS_INGRESS_SECRET');
    expect(()=>createHostedHttpServer(service,{allowedHost:'',ingressSecret})).toThrow('CUAN_SHEETS_MCP_ALLOWED_HOST');
  });
  it('requires exact Host, ingress secret, and one syntactically valid Connection Key before MCP handling',async()=>{
    const service={readValues:jest.fn(),updateValues:jest.fn()} as any;
    const server=createServer(service);
    await new Promise<void>(resolve=>server.listen(0,'127.0.0.1',resolve));
    const address=server.address(); if(!address||typeof address==='string') throw new Error('missing address');
    try {
      const endpoint=`http://127.0.0.1:${address.port}/mcp`,key=`ci_mcp_ck_${'b'.repeat(64)}`;
      const base={'host':allowedHost,'x-cuan-mcp-connection-key':key,'x-cuan-sheets-ingress-secret':ingressSecret};
      const wrongHost=await post(endpoint,{...base,host:'wrong.example'},'{}');
      expect(wrongHost.status).toBe(421);
      const missingIngress=await post(endpoint,{host:allowedHost,'x-cuan-mcp-connection-key':key},'{}');
      expect(missingIngress.status).toBe(401);
      const wrongIngress=await post(endpoint,{...base,'x-cuan-sheets-ingress-secret':'x'.repeat(40)},'{}');
      expect(wrongIngress.status).toBe(401);
      const missingKey=await post(endpoint,{host:allowedHost,'x-cuan-sheets-ingress-secret':ingressSecret},'{}');
      expect(missingKey.status).toBe(401);
      const invalid=await post(endpoint,{...base,'x-cuan-mcp-connection-key':'Bearer token'},'{}');
      expect(invalid.status).toBe(401);
      expect(service.readValues).not.toHaveBeenCalled();
    } finally { await new Promise<void>((resolve,reject)=>server.close(error=>error?reject(error):resolve())); }
  });
  it('serves only the two hosted Sheets tools over stateless Streamable HTTP',async()=>{
    const service={readValues:jest.fn(),updateValues:jest.fn()} as any;
    const server=createServer(service);
    await new Promise<void>(resolve=>server.listen(0,'127.0.0.1',resolve));
    const address=server.address(); if(!address||typeof address==='string') throw new Error('missing address');
    const endpoint=`http://127.0.0.1:${address.port}/mcp`,key=`ci_mcp_ck_${'b'.repeat(64)}`;
    const headers={'host':allowedHost,'content-type':'application/json',accept:'application/json, text/event-stream','x-cuan-mcp-connection-key':key,'x-cuan-sheets-ingress-secret':ingressSecret};
    try {
      const initialized=await post(endpoint,headers,JSON.stringify({jsonrpc:'2.0',id:1,method:'initialize',params:{protocolVersion:'2025-03-26',capabilities:{},clientInfo:{name:'test',version:'1'}}}));
      expect(initialized.status).toBe(200);
      const listed=await post(endpoint,headers,JSON.stringify({jsonrpc:'2.0',id:2,method:'tools/list',params:{}}));
      expect(listed.status).toBe(200);
      const payload=await listed.json() as {result:{tools:{name:string}[]}};
      expect(payload.result.tools.map(tool=>tool.name).sort()).toEqual(['sheets_read_values','sheets_update_values']);
    } finally { await new Promise<void>((resolve,reject)=>server.close(error=>error?reject(error):resolve())); }
  });
});
