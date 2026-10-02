/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { describe, expect, it, jest } from '@jest/globals';
import { CuanWorkspaceRuntimeClient } from '../../hostedWorkspace/CuanWorkspaceRuntimeClient';

describe('CuanWorkspaceRuntimeClient', () => {
  it('uses private Workspace service headers and v1 request', async () => {
    const fetchFn = jest.fn(async (_url: RequestInfo | URL, _init?: RequestInit) => new Response(JSON.stringify({ version: 1 }), { status: 200 }));
    const client = new CuanWorkspaceRuntimeClient('https://example.supabase.co/functions/v1/google-workspace-runtime', 'private-id', 'private-secret', fetchFn);
    const request = { version: 1, toolName: 'workspace_drive_list_files', targetId: 'account-root', executionId: 'x', requestDigest: 'a'.repeat(64) };
    expect(await client.invoke(`ci_mcp_ck_${'a'.repeat(64)}`, 'resolveRead', request)).toEqual({ version: 1 });
    expect(fetchFn.mock.calls[0][1]).toMatchObject({ method: 'POST', redirect: 'error', headers: { 'x-cuan-google-workspace-service-id': 'private-id', 'x-cuan-google-workspace-service-secret': 'private-secret' } });
    expect(JSON.parse(fetchFn.mock.calls[0][1].body as string)).toEqual({ action: 'resolveRead', request });
  });
  it('requires HTTPS and credentials', () => {
    expect(() => new CuanWorkspaceRuntimeClient('http://example.test', 'id', 'secret')).toThrow();
    expect(() => new CuanWorkspaceRuntimeClient('https://example.test', '', 'secret')).toThrow();
  });
  it('redeems a permit with service credentials and no caller key', async () => {
    const fetchFn = jest.fn(async (_url: RequestInfo | URL, _init?: RequestInit) => new Response(JSON.stringify({ ok: true, accessToken: 'transient' }), { status: 200 }));
    const client = new CuanWorkspaceRuntimeClient('https://example.supabase.co/functions/v1/google-workspace-runtime', 'private-id', 'private-secret', fetchFn,
      'https://example.supabase.co/functions/v1/mcp-redeem-google-permit');
    const googleInvocation = { version: 1, publicTool: 'workspace_drive_list_files', permit: 'opaque' };
    expect(await client.redeem(googleInvocation)).toMatchObject({ ok: true, accessToken: 'transient' });
    expect(fetchFn.mock.calls[0][1].headers).not.toHaveProperty('x-cuan-mcp-connection-key');
    expect(JSON.parse(fetchFn.mock.calls[0][1].body as string)).toEqual({ googleInvocation });
    await client.finalize({ executionId: 'execution_123', permit: 'opaque' }, 'succeeded');
    expect(fetchFn.mock.calls[1][0]).toBe('https://example.supabase.co/functions/v1/mcp-finalize-execution');
    expect(JSON.parse(fetchFn.mock.calls[1][1].body as string)).toEqual({ executionId: 'execution_123', permit: 'opaque', outcome: 'succeeded' });
  });
});
