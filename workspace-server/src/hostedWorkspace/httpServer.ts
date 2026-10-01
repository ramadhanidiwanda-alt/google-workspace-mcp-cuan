/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { timingSafeEqual } from 'node:crypto';
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StreamableHTTPServerTransport } from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import type { CallToolResult } from '@modelcontextprotocol/sdk/types.js';
import { z } from 'zod';
import { GoogleWorkspaceHostedService, HostedWorkspaceError } from './GoogleWorkspaceHostedService';

const KEY = /^ci_mcp_ck_[0-9a-f]{64}$/;
const HOST = 'host', INGRESS = 'x-cuan-workspace-ingress-secret', CONNECTION_KEY = 'x-cuan-mcp-connection-key';
const cells = z.array(z.array(z.union([z.string(), z.number(), z.boolean()])));
const confirmation = { previewId: z.string().optional(), approvalDigest: z.string().optional(), confirmed: z.boolean().optional() };
export interface HostedWorkspaceHttpBoundary { allowedHost: string; ingressSecret: string }

function rawHeader(req: IncomingMessage, name: string): string[] {
  const values: string[] = [];
  for (let i = 0; i < req.rawHeaders.length; i += 2) if (req.rawHeaders[i].toLowerCase() === name) values.push(req.rawHeaders[i + 1]);
  return values;
}
function validIngress(req: IncomingMessage, expected: string) {
  const values = rawHeader(req, INGRESS);
  if (values.length !== 1) return false;
  const actual = Buffer.from(values[0]), wanted = Buffer.from(expected);
  return actual.length === wanted.length && timingSafeEqual(actual, wanted);
}
function fail(res: ServerResponse, status: number, code: string) {
  res.writeHead(status, { 'content-type': 'application/json', 'cache-control': 'no-store' });
  res.end(JSON.stringify({ error: code }));
}
async function readBody(req: IncomingMessage): Promise<unknown> {
  const chunks: Buffer[] = []; let size = 0;
  for await (const chunk of req) {
    const buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
    size += buffer.length;
    if (size > 262144) throw new Error('REQUEST_TOO_LARGE');
    chunks.push(buffer);
  }
  return JSON.parse(Buffer.concat(chunks).toString('utf8'));
}
function createMcp(key: string, service: GoogleWorkspaceHostedService) {
  const server = new McpServer({ name: 'cuan-google-workspace', version: '1.0.0' });
  const register = <T extends Record<string, z.ZodTypeAny>>(name: string, description: string, inputSchema: T, run: (args: z.infer<z.ZodObject<T>>) => Promise<unknown>) => {
    const registerTool = server.registerTool.bind(server) as unknown as (toolName: string, config: { description: string; inputSchema: T }, callback: (args: z.infer<z.ZodObject<T>>) => Promise<CallToolResult>) => void;
    registerTool(name, { description, inputSchema }, async (args): Promise<CallToolResult> => {
      try {
        const result = await run(args as z.infer<z.ZodObject<T>>);
        const output = JSON.stringify(result);
        if (Buffer.byteLength(output) > 131072) throw new HostedWorkspaceError('OUTPUT_TOO_LARGE');
        return { content: [{ type: 'text' as const, text: output }] };
      } catch (error) {
        return { isError: true, content: [{ type: 'text' as const, text: error instanceof HostedWorkspaceError ? error.code : 'WORKSPACE_TOOL_FAILED' }] };
      }
    });
  };
  register('workspace_drive_list_files', 'List up to 100 files accessible to the connected Google account.', { pageSize: z.number().int().min(1).max(100).optional(), pageToken: z.string().optional() }, args => service.listFiles(key, args));
  register('workspace_drive_get_file', 'Read Drive file metadata.', { fileId: z.string() }, args => service.getFile(key, args));
  register('workspace_drive_get_text', 'Read bounded plain text or Google Docs text.', { fileId: z.string() }, args => service.getText(key, args));
  register('workspace_sheets_read_values', 'Read one bounded Google Sheets range.', { spreadsheetId: z.string(), range: z.string() }, args => service.readValues(key, args));
  register('workspace_drive_create_text_file', 'Preview and confirm creation of a plain text Drive file.', { name: z.string(), content: z.string(), parentId: z.string().optional(), ...confirmation }, args => service.createTextFile(key, args));
  register('workspace_drive_update_text_file', 'Preview and confirm replacement of an existing plain text Drive file.', { fileId: z.string(), expectedOldText: z.string(), newText: z.string(), ...confirmation }, args => service.updateTextFile(key, args));
  register('workspace_sheets_update_values', 'Preview and confirm a bounded Sheets update.', { spreadsheetId: z.string(), range: z.string(), expectedOldValues: cells, newValues: cells, ...confirmation }, args => service.updateValues(key, args));
  return server;
}
export function createHostedWorkspaceHttpServer(service: GoogleWorkspaceHostedService, boundary: HostedWorkspaceHttpBoundary) {
  if (!/^[a-z0-9.-]+$/.test(boundary.allowedHost)) throw new Error('CUAN_WORKSPACE_MCP_ALLOWED_HOST must be a canonical hostname');
  if (boundary.ingressSecret.length < 32) throw new Error('CUAN_WORKSPACE_INGRESS_SECRET must contain at least 32 characters');
  return createServer(async (req, res) => {
    const hosts = rawHeader(req, HOST);
    if (hosts.length !== 1 || hosts[0] !== boundary.allowedHost) { fail(res, 421, 'MISDIRECTED_REQUEST'); return; }
    if (req.url !== '/mcp') { fail(res, 404, 'NOT_FOUND'); return; }
    if (req.method !== 'POST') { fail(res, 405, 'METHOD_NOT_ALLOWED'); return; }
    if (!validIngress(req, boundary.ingressSecret)) { fail(res, 401, 'UNAUTHENTICATED'); return; }
    const keys = rawHeader(req, CONNECTION_KEY);
    if (keys.length !== 1 || !KEY.test(keys[0])) { fail(res, 401, 'UNAUTHENTICATED'); return; }
    if (Number(req.headers['content-length'] ?? 0) > 262144) { fail(res, 413, 'REQUEST_TOO_LARGE'); return; }
    if (!/^application\/json(?:\s*;|$)/i.test(req.headers['content-type'] ?? '')) { fail(res, 415, 'UNSUPPORTED_MEDIA_TYPE'); return; }
    let body: unknown;
    try { body = await readBody(req); }
    catch (error) { fail(res, error instanceof Error && error.message === 'REQUEST_TOO_LARGE' ? 413 : 400, error instanceof Error && error.message === 'REQUEST_TOO_LARGE' ? 'REQUEST_TOO_LARGE' : 'INVALID_JSON'); return; }
    const server = createMcp(keys[0], service);
    const transport = new StreamableHTTPServerTransport({ sessionIdGenerator: undefined, enableJsonResponse: true });
    res.once('finish', () => { void transport.close(); void server.close(); });
    try { await server.connect(transport); await transport.handleRequest(req, res, body); }
    catch { if (!res.headersSent) fail(res, 400, 'INVALID_MCP_REQUEST'); }
    finally { if (res.writableEnded) { void transport.close(); void server.close(); } }
  });
}
export function startHostedWorkspaceHttpServer(service: GoogleWorkspaceHostedService, host = '0.0.0.0', port = 8080) {
  const boundary = { allowedHost: process.env.CUAN_WORKSPACE_MCP_ALLOWED_HOST ?? '', ingressSecret: process.env.CUAN_WORKSPACE_INGRESS_SECRET ?? '' };
  const server = createHostedWorkspaceHttpServer(service, boundary);
  server.listen(port, host);
  return server;
}
