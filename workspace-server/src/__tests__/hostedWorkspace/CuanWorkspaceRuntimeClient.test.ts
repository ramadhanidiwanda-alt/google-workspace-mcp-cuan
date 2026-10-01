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
});
