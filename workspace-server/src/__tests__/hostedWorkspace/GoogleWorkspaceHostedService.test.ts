/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { describe, expect, it, jest } from '@jest/globals';
import { GoogleWorkspaceHostedService } from '../../hostedWorkspace/GoogleWorkspaceHostedService';
import type { WorkspaceRuntime } from '../../hostedWorkspace/CuanWorkspaceRuntimeClient';

const key = `ci_mcp_ck_${'a'.repeat(64)}`;
const id = 'a'.repeat(28);
const now = 1_800_000_000_000;
const token = 'sensitive-access-token';
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });
const plain = (body: string) => new Response(body, { status: 200, headers: { 'content-type': 'text/plain' } });
function runtime(override?: (action: string, request: Record<string, unknown>) => Record<string, unknown>) {
  const invoke = jest.fn(async (_key: string, action: string, request: Record<string, unknown>) => {
    if (override) return override(action, request);
    return { ...request, accountId: id, credentialVersion: 1, expiresAt: now + 60000, accessToken: token, ...(action === 'claimWrite' ? { writeClaimed: true } : {}), ...(action === 'finalizeWrite' ? { finalized: true } : {}) };
  });
  return { invoke } as WorkspaceRuntime & { invoke: typeof invoke };
}

describe('GoogleWorkspaceHostedService', () => {
  it('binds every read to Cuan and redacts credentials from output', async () => {
    const auth = runtime();
    const fetchFn = jest.fn(async (_url: RequestInfo | URL, init?: RequestInit) => {
      expect(init?.headers).toMatchObject({ authorization: `Bearer ${token}` });
      return json({ files: [{ id, name: 'Plan', mimeType: 'text/plain' }] });
    });
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    const result = await service.listFiles(key, { pageSize: 10 });
    expect(result.files).toHaveLength(1);
    expect(JSON.stringify(result)).not.toContain(token);
    expect(auth.invoke.mock.calls[0][2]).toMatchObject({ version: 1, toolName: 'workspace_drive_list_files', targetId: 'account-root', requestDigest: expect.stringMatching(/^[0-9a-f]{64}$/) });
  });

  it('rejects a mismatched Cuan binding before contacting Google', async () => {
    const auth = runtime((_action, request) => ({ ...request, targetId: 'wrong', accountId: id, credentialVersion: 1, expiresAt: now + 60000, accessToken: token }));
    const fetchFn = jest.fn(async () => json({}));
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    await expect(service.getFile(key, { fileId: id })).rejects.toMatchObject({ code: 'CUAN_BINDING_MISMATCH' });
    expect(fetchFn).not.toHaveBeenCalled();
  });

  it('accepts a bounded Cuan credential lease shorter than ten minutes', async () => {
    const auth = runtime((_action, request) => ({ ...request, accountId: id, credentialVersion: 1, expiresAt: now + 9 * 60_000, accessToken: token }));
    const fetchFn = jest.fn(async () => json({ id, name: 'note', mimeType: 'text/plain' }));
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    expect(await service.getFile(key, { fileId: id })).toMatchObject({ file: { id } });
    expect(fetchFn).toHaveBeenCalledTimes(1);
  });

  it('reads only text MIME content', async () => {
    const auth = runtime();
    const fetchFn = jest.fn(async (url: RequestInfo | URL) => String(url).includes('alt=media') ? plain('hello') : json({ id, name: 'note', mimeType: 'text/plain' }));
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    expect(await service.getText(key, { fileId: id })).toEqual({ fileId: id, name: 'note', mimeType: 'text/plain', content: 'hello' });
    expect(fetchFn).toHaveBeenCalledTimes(2);
  });

  it('previews, checks current text, claims once, and finalizes a confirmed update', async () => {
    const auth = runtime();
    let reads = 0, written = false;
    const fetchFn = jest.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'PATCH') { written = true; return json({ id, name: 'note', mimeType: 'text/plain' }); }
      if (String(url).includes('alt=media')) { reads++; return plain(written ? 'after' : 'before'); }
      return json({ id, name: 'note', mimeType: 'text/plain' });
    });
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    const preview = await service.updateTextFile(key, { fileId: id, expectedOldText: 'before', newText: 'after' });
    expect(preview).toMatchObject({ requiresConfirmation: true, before: 'before' });
    const result = await service.updateTextFile(key, { fileId: id, expectedOldText: 'before', newText: 'after', confirmed: true, previewId: preview.previewId as string, approvalDigest: preview.approvalDigest as string });
    expect(result).toMatchObject({ status: 'CONFIRMED' });
    expect(reads).toBe(3);
    expect(auth.invoke.mock.calls.map(([, action]) => action)).toEqual(['resolveRead', 'resolveRead', 'resolveRead', 'claimWrite', 'finalizeWrite']);
    expect(auth.invoke.mock.calls[3][2]).toMatchObject({ previewId: preview.previewId, toolName: 'workspace_drive_update_text_file', targetId: id, requestDigest: preview.approvalDigest });
    expect(fetchFn.mock.calls.filter(([, init]) => init?.method === 'PATCH')).toHaveLength(1);
  });

  it('finalizes unknown outcome after one dispatched write and never retries', async () => {
    const auth = runtime();
    const fetchFn = jest.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'PATCH') throw new Error('socket closed');
      return String(url).includes('alt=media') ? plain('before') : json({ id, name: 'note', mimeType: 'text/plain' });
    });
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    const preview = await service.updateTextFile(key, { fileId: id, expectedOldText: 'before', newText: 'after' });
    const result = await service.updateTextFile(key, { fileId: id, expectedOldText: 'before', newText: 'after', confirmed: true, previewId: preview.previewId as string, approvalDigest: preview.approvalDigest as string });
    expect(result).toMatchObject({ status: 'UNKNOWN_OUTCOME' });
    expect(fetchFn.mock.calls.filter(([, init]) => init?.method === 'PATCH')).toHaveLength(1);
    expect(auth.invoke.mock.calls[auth.invoke.mock.calls.length - 1][2]).toMatchObject({ status: 'UNKNOWN_OUTCOME' });
  });

  it('creates a text file only after confirmation and binds account-root', async () => {
    const auth = runtime();
    const fetchFn = jest.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'POST') { expect(init?.body).toContain('hello'); return json({ id, name: 'note', mimeType: 'text/plain' }); }
      return String(url).includes('alt=media') ? plain('hello') : json({ id, name: 'note', mimeType: 'text/plain' });
    });
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    const preview = await service.createTextFile(key, { name: 'note', content: 'hello' });
    expect(fetchFn).not.toHaveBeenCalled();
    await expect(service.createTextFile(key, { name: 'note', content: 'changed', confirmed: true, previewId: preview.previewId as string, approvalDigest: preview.approvalDigest as string }))
      .rejects.toMatchObject({ code: 'CONFIRMATION_REQUIRED' });
    const result = await service.createTextFile(key, { name: 'note', content: 'hello', confirmed: true, previewId: preview.previewId as string, approvalDigest: preview.approvalDigest as string });
    expect(result).toMatchObject({ status: 'CONFIRMED', result: { fileId: id } });
    expect(fetchFn).toHaveBeenCalledTimes(3);
    expect(fetchFn.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1);
    expect(auth.invoke.mock.calls[1][2]).toMatchObject({ toolName: 'workspace_drive_create_text_file', targetId: 'account-root', preview: true,
      previewSummary: { operation: 'workspace_drive_create_text_file', targetId: 'account-root', name: 'note', beforeExcerpt: '', afterExcerpt: 'hello', beforeLength: 0, afterLength: 5 } });
  });

  it('blocks stale Sheets values before claim or mutation', async () => {
    const auth = runtime();
    const fetchFn = jest.fn(async (_url: RequestInfo | URL, _init?: RequestInit) => json({ values: [['other']] }));
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    await expect(service.updateValues(key, { spreadsheetId: id, range: "'Data'!A1:A1", expectedOldValues: [['old']], newValues: [['new']] }))
      .rejects.toMatchObject({ code: 'STALE_CONTENT' });
    expect(auth.invoke.mock.calls.map(([, action]) => action)).toEqual(['resolveRead']);
    expect(fetchFn.mock.calls.every(([, init]) => init?.method !== 'PUT')).toBe(true);
  });

  it('normalizes trimmed empty Sheets cells and confirms one RAW update', async () => {
    const auth = runtime();
    let written = false;
    const fetchFn = jest.fn(async (_url: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'PUT') { written = true; return json({ updatedCells: 2 }); }
      return json({ values: written ? [['one', 'two']] : [] });
    });
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    const input = { spreadsheetId: id, range: "'Data'!A1:B1", expectedOldValues: [['', '']], newValues: [['one', 'two']] };
    const preview = await service.updateValues(key, input);
    expect(preview).toMatchObject({ before: [['', '']], requiresConfirmation: true });
    const result = await service.updateValues(key, { ...input, confirmed: true, previewId: preview.previewId as string, approvalDigest: preview.approvalDigest as string });
    expect(result).toMatchObject({ status: 'CONFIRMED' });
    expect(fetchFn.mock.calls.filter(([, init]) => init?.method === 'PUT')).toHaveLength(1);
  });

  it('finalizes unknown when a successful Drive update reads back stale bytes', async () => {
    const auth = runtime();
    const fetchFn = jest.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'PATCH') return json({ id, mimeType: 'text/plain' });
      return String(url).includes('alt=media') ? plain('before') : json({ id, name: 'note', mimeType: 'text/plain' });
    });
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    const input = { fileId: id, expectedOldText: 'before', newText: 'after' };
    const preview = await service.updateTextFile(key, input);
    const result = await service.updateTextFile(key, { ...input, confirmed: true, previewId: preview.previewId as string, approvalDigest: preview.approvalDigest as string });
    expect(result).toMatchObject({ status: 'UNKNOWN_OUTCOME' });
    expect(fetchFn.mock.calls.filter(([, init]) => init?.method === 'PATCH')).toHaveLength(1);
    expect(auth.invoke.mock.calls[auth.invoke.mock.calls.length - 1][2]).toMatchObject({ status: 'UNKNOWN_OUTCOME' });
  });

  it('finalizes unknown when a created Drive file reads back different bytes', async () => {
    const auth = runtime();
    const fetchFn = jest.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'POST') return json({ id, name: 'note', mimeType: 'text/plain' });
      return String(url).includes('alt=media') ? plain('different') : json({ id, name: 'note', mimeType: 'text/plain' });
    });
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    const input = { name: 'note', content: 'expected' };
    const preview = await service.createTextFile(key, input);
    const result = await service.createTextFile(key, { ...input, confirmed: true, previewId: preview.previewId as string, approvalDigest: preview.approvalDigest as string });
    expect(result).toMatchObject({ status: 'UNKNOWN_OUTCOME' });
    expect(fetchFn.mock.calls.filter(([, init]) => init?.method === 'POST')).toHaveLength(1);
    expect(auth.invoke.mock.calls[auth.invoke.mock.calls.length - 1][2]).toMatchObject({ status: 'UNKNOWN_OUTCOME' });
  });

  it('finalizes unknown when a successful Sheets update reads back stale cells', async () => {
    const auth = runtime();
    const fetchFn = jest.fn(async (_url: RequestInfo | URL, init?: RequestInit) => init?.method === 'PUT' ? json({ updatedCells: 1 }) : json({ values: [['old']] }));
    const service = new GoogleWorkspaceHostedService({ runtime: auth, fetchFn, now: () => now });
    const input = { spreadsheetId: id, range: "'Data'!A1:A1", expectedOldValues: [['old']], newValues: [['new']] };
    const preview = await service.updateValues(key, input);
    const result = await service.updateValues(key, { ...input, confirmed: true, previewId: preview.previewId as string, approvalDigest: preview.approvalDigest as string });
    expect(result).toMatchObject({ status: 'UNKNOWN_OUTCOME' });
    expect(fetchFn.mock.calls.filter(([, init]) => init?.method === 'PUT')).toHaveLength(1);
    expect(auth.invoke.mock.calls[auth.invoke.mock.calls.length - 1][2]).toMatchObject({ status: 'UNKNOWN_OUTCOME' });
  });
});
