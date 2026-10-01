/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { describe, expect, it, jest } from '@jest/globals';
import { request as httpRequest } from 'node:http';
import { createHostedWorkspaceHttpServer } from '../../hostedWorkspace/httpServer';

const host = 'workspace-mcp.cuaninsight.com', ingressSecret = 'i'.repeat(40), key = `ci_mcp_ck_${'b'.repeat(64)}`;
function post(endpoint: string, headers: Record<string, string>, body: string) {
  return new Promise<{ status: number; json: () => Promise<any> }>((resolve, reject) => {
    const request = httpRequest(endpoint, { method: 'POST', headers }, response => {
      const chunks: Buffer[] = [];
      response.on('data', chunk => chunks.push(Buffer.from(chunk)));
      response.on('end', () => resolve({ status: response.statusCode ?? 0, json: async () => JSON.parse(Buffer.concat(chunks).toString('utf8')) }));
    });
    request.on('error', reject);
    request.end(body);
  });
}
describe('hosted Workspace HTTP boundary', () => {
  it('requires exact Host, ingress secret, and key before MCP handling', async () => {
    const service = { listFiles: jest.fn() } as any;
    expect(() => createHostedWorkspaceHttpServer(service, { allowedHost: '', ingressSecret })).toThrow('CUAN_WORKSPACE_MCP_ALLOWED_HOST');
    expect(() => createHostedWorkspaceHttpServer(service, { allowedHost: host, ingressSecret: 'short' })).toThrow('CUAN_WORKSPACE_INGRESS_SECRET');
    const server = createHostedWorkspaceHttpServer(service, { allowedHost: host, ingressSecret });
    await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve));
    const address = server.address(); if (!address || typeof address === 'string') throw new Error('missing address');
    const endpoint = `http://127.0.0.1:${address.port}/mcp`;
    try {
      const base = { host, 'x-cuan-workspace-ingress-secret': ingressSecret, 'x-cuan-mcp-connection-key': key, 'content-type': 'application/json' };
      expect((await post(endpoint, { ...base, host: 'other.example' }, '{}')).status).toBe(421);
      expect((await post(endpoint, { ...base, 'x-cuan-workspace-ingress-secret': 'x'.repeat(40) }, '{}')).status).toBe(401);
      expect((await post(endpoint, { ...base, 'x-cuan-mcp-connection-key': 'Bearer token' }, '{}')).status).toBe(401);
      expect(service.listFiles).not.toHaveBeenCalled();
    } finally { await new Promise<void>(resolve => server.close(() => resolve())); }
  });
  it('publishes only seven broad Workspace tools', async () => {
    const server = createHostedWorkspaceHttpServer({} as any, { allowedHost: host, ingressSecret });
    await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve));
    const address = server.address(); if (!address || typeof address === 'string') throw new Error('missing address');
    const endpoint = `http://127.0.0.1:${address.port}/mcp`;
    const headers = { host, 'x-cuan-workspace-ingress-secret': ingressSecret, 'x-cuan-mcp-connection-key': key, 'content-type': 'application/json', accept: 'application/json, text/event-stream' };
    try {
      const response = await post(endpoint, headers, JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'tools/list', params: {} }));
      expect(response.status).toBe(200);
      const body = await response.json();
      expect(body.result.tools.map((tool: { name: string }) => tool.name).sort()).toEqual([
        'workspace_drive_create_text_file', 'workspace_drive_get_file', 'workspace_drive_get_text', 'workspace_drive_list_files',
        'workspace_drive_update_text_file', 'workspace_sheets_read_values', 'workspace_sheets_update_values',
      ]);
    } finally { await new Promise<void>(resolve => server.close(() => resolve())); }
  });
  it('routes a bounded MCP tool call with the connection key', async () => {
    const service = { listFiles: jest.fn(async () => ({ files: [{ id: 'sample' }] })) } as any;
    const server = createHostedWorkspaceHttpServer(service, { allowedHost: host, ingressSecret });
    await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve));
    const address = server.address(); if (!address || typeof address === 'string') throw new Error('missing address');
    const endpoint = `http://127.0.0.1:${address.port}/mcp`;
    const headers = { host, 'x-cuan-workspace-ingress-secret': ingressSecret, 'x-cuan-mcp-connection-key': key, 'content-type': 'application/json', accept: 'application/json, text/event-stream' };
    try {
      const response = await post(endpoint, headers, JSON.stringify({ jsonrpc: '2.0', id: 2, method: 'tools/call', params: { name: 'workspace_drive_list_files', arguments: { pageSize: 5 } } }));
      expect(response.status).toBe(200);
      expect(await response.json()).toMatchObject({ result: { content: [{ type: 'text', text: '{"files":[{"id":"sample"}]}' }] } });
      expect(service.listFiles).toHaveBeenCalledWith(key, { pageSize: 5 });
    } finally { await new Promise<void>(resolve => server.close(() => resolve())); }
  });
});
