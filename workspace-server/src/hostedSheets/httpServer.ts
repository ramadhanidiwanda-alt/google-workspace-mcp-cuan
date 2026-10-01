/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StreamableHTTPServerTransport } from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import { z } from 'zod';
import { GoogleSheetsHostedService, HostedSheetsError } from './GoogleSheetsHostedService';

const keyName='x-cuan-mcp-connection-key';
function rawKey(req:IncomingMessage):string|undefined {
  const found:string[]=[]; for(let i=0;i<req.rawHeaders.length;i+=2) if(req.rawHeaders[i].toLowerCase()===keyName) found.push(req.rawHeaders[i+1]);
  return found.length===1 && /^ci_mcp_ck_[0-9a-f]{64}$/.test(found[0]) ? found[0] : undefined;
}
function fail(res:ServerResponse,status:number,code:string) { res.writeHead(status,{'content-type':'application/json','cache-control':'no-store'});res.end(JSON.stringify({error:code})); }
async function readBody(req:IncomingMessage):Promise<unknown> {
  const chunks:Buffer[]=[];let size=0;
  for await (const chunk of req) { const buffer=Buffer.isBuffer(chunk)?chunk:Buffer.from(chunk);size+=buffer.length;if(size>32768)throw new Error('REQUEST_TOO_LARGE');chunks.push(buffer); }
  if(size===0) return undefined;
  return JSON.parse(Buffer.concat(chunks).toString('utf8'));
}
function createMcp(key:string,service:GoogleSheetsHostedService) {
  const server=new McpServer({name:'cuan-google-sheets',version:'1.0.0'});
  server.registerTool('sheets_read_values',{description:'Read values from one exact, bounded Google Sheets range authorized by Cuan.',inputSchema:{spreadsheetId:z.string(),range:z.string()}},async ({spreadsheetId,range})=>{
    try { const value=await service.readValues(key,{spreadsheetId,range});return {content:[{type:'text',text:JSON.stringify(value)}]}; }
    catch(e){return {isError:true,content:[{type:'text',text:e instanceof HostedSheetsError?e.code:'SHEETS_READ_FAILED'}]};}
  });
  server.registerTool('sheets_update_values',{description:'Preview a bounded value update, then repeat with the returned IDs and digest plus confirmed=true to request a Cuan policy-gated write.',inputSchema:{spreadsheetId:z.string(),range:z.string(),expectedOldValues:z.array(z.array(z.union([z.string(),z.number(),z.boolean()]))),newValues:z.array(z.array(z.union([z.string(),z.number(),z.boolean()]))),previewId:z.string().optional(),executionId:z.string().optional(),approvalDigest:z.string().optional(),confirmed:z.boolean().optional()}},async (args)=>{
    try { const value=await service.updateValues(key,args);return {content:[{type:'text',text:JSON.stringify(value)}]}; }
    catch(e){return {isError:true,content:[{type:'text',text:e instanceof HostedSheetsError?e.code:'SHEETS_UPDATE_FAILED'}]};}
  });
  return server;
}
export function createHostedHttpServer(service:GoogleSheetsHostedService) {
  return createServer(async (req,res)=>{
    if(req.url!=='/mcp'){fail(res,404,'NOT_FOUND');return;}
    if(req.method!=='POST'){fail(res,405,'METHOD_NOT_ALLOWED');return;}
    const key=rawKey(req);if(!key){fail(res,401,'UNAUTHENTICATED');return;}
    const contentLength=Number(req.headers['content-length'] ?? 0);
    if(contentLength>32768){fail(res,413,'REQUEST_TOO_LARGE');return;}
    if(!/^application\/json(?:\s*;|$)/i.test(req.headers['content-type']??'')){fail(res,415,'UNSUPPORTED_MEDIA_TYPE');return;}
    let body:unknown;
    try { body=await readBody(req); }
    catch(e){fail(res,e instanceof Error&&e.message==='REQUEST_TOO_LARGE'?413:400,e instanceof Error&&e.message==='REQUEST_TOO_LARGE'?'REQUEST_TOO_LARGE':'INVALID_JSON');return;}
    const server=createMcp(key,service);
    const transport=new StreamableHTTPServerTransport({sessionIdGenerator:undefined,enableJsonResponse:true});
    res.once('finish',()=>{void transport.close();void server.close();});
    try { await server.connect(transport); await transport.handleRequest(req,res,body); }
    catch { if(!res.headersSent) fail(res,400,'INVALID_MCP_REQUEST'); }
    finally { if(res.writableEnded){void transport.close();void server.close();} }
  });
}
export function startHostedHttpServer(service:GoogleSheetsHostedService,host='0.0.0.0',port=8080) {
  const server=createHostedHttpServer(service); server.listen(port,host); return server;
}
