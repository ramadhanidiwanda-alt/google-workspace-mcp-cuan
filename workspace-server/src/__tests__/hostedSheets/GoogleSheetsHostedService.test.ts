/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { describe, expect, it, jest } from '@jest/globals';
import { GoogleSheetsHostedService } from '../../hostedSheets/GoogleSheetsHostedService';
import type { CuanSheetsRuntime } from '../../hostedSheets/CuanSheetsRuntimeClient';

const connectionKey = `ci_mcp_ck_${'a'.repeat(64)}`;
const spreadsheetId = '1'.repeat(25);
const range = "'Data Sheet'!A1:B2";
const now = 1_800_000_000_000;
const validCredential = (request: Record<string, unknown>, writeClaimed = false) => ({
  accessToken: 'provider-access-token-sentinel',
  allowedRange: request.range,
  expiresAt: now + 60_000,
  executionId: request.executionId,
  requestDigest: request.requestDigest,
  scope: 'https://www.googleapis.com/auth/spreadsheets',
  spreadsheetId: request.spreadsheetId,
  ...(writeClaimed ? { writeClaimed: true } : {}),
});
const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), {
  status,
  headers: { 'content-type': 'application/json' },
});
const makeRuntime = (invoke: CuanSheetsRuntime['invoke']) => ({ invoke: jest.fn(invoke) });

describe('GoogleSheetsHostedService', () => {
  it('resolves each bounded read through Cuan and returns only normalized cell values', async () => {
    const runtime = makeRuntime(async (_key, action, request) => {
      expect(action).toBe('resolveRead');
      expect(request).toMatchObject({ operation: 'read_values', spreadsheetId, range });
      return validCredential(request);
    });
    const fetchFn = jest.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      expect(init?.headers).toMatchObject({ authorization: 'Bearer provider-access-token-sentinel' });
      expect(init?.redirect).toBe('error');
      return response({ values: [['A', 7], [true]] });
    });
    const service = new GoogleSheetsHostedService({ runtime, fetchFn, now: () => now });

    const result = await service.readValues(connectionKey, { spreadsheetId, range });

    expect(result).toEqual({ spreadsheetId, range, values: [['A', 7], [true, '']] });
    expect(JSON.stringify(result)).not.toContain('provider-access-token-sentinel');
    expect(fetchFn).toHaveBeenCalledTimes(1);
  });

  it('rejects broad, malformed, oversized, or unquoted ranges before contacting Cuan', async () => {
    const runtime = makeRuntime(async () => { throw new Error('should not be called'); });
    const service = new GoogleSheetsHostedService({ runtime, fetchFn: fetch, now: () => now });
    for (const invalidRange of ['A:Z', "'Data Sheet'!A1:Z20", "'Data Sheet'!A1:A101", 'Data!A1:B2']) {
      await expect(service.readValues(connectionKey, { spreadsheetId, range: invalidRange })).rejects.toMatchObject({ code: 'INVALID_TARGET' });
    }
    expect(runtime.invoke).not.toHaveBeenCalled();
  });

  it('fails closed on invalid Connection Keys without provider requests', async () => {
    const runtime = makeRuntime(async () => { throw new Error('should not be called'); });
    const fetchFn = jest.fn(async () => response({ values: [] }));
    const service = new GoogleSheetsHostedService({ runtime, fetchFn, now: () => now });
    await expect(service.readValues('Bearer some-token', { spreadsheetId, range })).rejects.toMatchObject({ code: 'UNAUTHENTICATED' });
    expect(runtime.invoke).not.toHaveBeenCalled();
    expect(fetchFn).not.toHaveBeenCalled();
  });

  it('returns a no-write preview bound to exact old and new cells', async () => {
    const oldValues = [['old', 'x'], ['y', 1]];
    const newValues = [['new', 'x'], ['y', 2]];
    const runtime = makeRuntime(async (_key, action, request) => {
      expect(action).toBe('resolveRead');
      expect(request).toMatchObject({ operation: 'preview_update', spreadsheetId, range });
      return validCredential(request);
    });
    const fetchFn = jest.fn(async () => response({ values: oldValues }));
    const service = new GoogleSheetsHostedService({ runtime, fetchFn, now: () => now });

    const preview = await service.updateValues(connectionKey, { spreadsheetId, range, expectedOldValues: oldValues, newValues });

    expect(preview).toMatchObject({ spreadsheetId, range, oldValues, newValues, requiresConfirmation: true });
    expect(preview.previewId).toMatch(/^[0-9a-f-]{36}$/);
    expect(preview.executionId).toMatch(/^[0-9a-f-]{36}$/);
    expect(preview.approvalDigest).toMatch(/^[0-9a-f]{64}$/);
    expect(fetchFn).toHaveBeenCalledTimes(1);
    expect(fetchFn.mock.calls[0][1]?.method).toBe('GET');
    expect(runtime.invoke).toHaveBeenCalledTimes(1);
  });

  it('blocks stale or formula-bearing previews and never claims a write', async () => {
    const oldValues = [['old', 'x'], ['y', 1]];
    const runtime = makeRuntime(async (_key, _action, request) => validCredential(request));
    const fetchFn = jest.fn(async () => response({ values: [['stale', 'x'], ['y', 1]] }));
    const service = new GoogleSheetsHostedService({ runtime, fetchFn, now: () => now });
    await expect(service.updateValues(connectionKey, { spreadsheetId, range, expectedOldValues: oldValues, newValues: [['new', 'x'], ['y', 2]] }))
      .rejects.toMatchObject({ code: 'STALE_VALUES' });
    await expect(service.updateValues(connectionKey, { spreadsheetId, range, expectedOldValues: [['=SUM(A1:A2)', 'x'], ['y', 1]], newValues: [['new', 'x'], ['y', 2]] }))
      .rejects.toMatchObject({ code: 'FORMULA_PROTECTED' });
    expect(runtime.invoke.mock.calls.some(([, action]) => action === 'claimWrite')).toBe(false);
    expect(fetchFn.mock.calls.every(([, init]) => init?.method !== 'PUT')).toBe(true);
  });

  it('requires exact preview confirmation, claims once, verifies the write, then finalizes exact binding', async () => {
    const oldValues = [['old', 'x'], ['y', 1]];
    const newValues = [['new', 'x'], ['y', 2]];
    let reads = 0;
    const runtime = makeRuntime(async (_key, action, request) => {
      if (action === 'claimWrite') return validCredential(request, true);
      if (action === 'finalizeWrite') return { ...request, finalized: true, operation: 'update_values' };
      expect(action).toBe('resolveRead');
      expect(request).toMatchObject({ operation: expect.stringMatching(/^(preview_update|confirm_update)$/), executionId: expect.any(String), requestDigest: expect.any(String) });
      return validCredential(request);
    });
    const fetchFn = jest.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'PUT') return response({ updatedCells: 4 });
      reads++;
      return response({ values: reads < 3 ? oldValues : newValues });
    });
    const service = new GoogleSheetsHostedService({ runtime, fetchFn, now: () => now });
    const preview = await service.updateValues(connectionKey, { spreadsheetId, range, expectedOldValues: oldValues, newValues });

    const result = await service.updateValues(connectionKey, {
      spreadsheetId, range, expectedOldValues: oldValues, newValues,
      previewId: preview.previewId, executionId: preview.executionId, approvalDigest: preview.approvalDigest, confirmed: true,
    });

    expect(result).toMatchObject({ status: 'CONFIRMED', spreadsheetId, range, executionId: preview.executionId });
    expect(runtime.invoke.mock.calls.map(([, action]) => action)).toEqual(['resolveRead', 'resolveRead', 'claimWrite', 'finalizeWrite']);
    const finalRequest = runtime.invoke.mock.calls[3][2];
    expect(finalRequest).toMatchObject({ operation: 'update_values', executionId: preview.executionId, spreadsheetId, range, status: 'CONFIRMED' });
    const put = fetchFn.mock.calls.find(([, init]) => init?.method === 'PUT');
    expect(put?.[1]?.body).toContain('"values"');
    expect(put?.[1]?.headers).toMatchObject({ authorization: 'Bearer provider-access-token-sentinel' });
  });

  it('requires caller confirmation and the exact digest before any claim or write', async () => {
    const oldValues = [['old', 'x'], ['y', 1]], newValues = [['new', 'x'], ['y', 2]];
    const runtime = makeRuntime(async (_key, _action, request) => validCredential(request));
    const fetchFn = jest.fn(async () => response({ values: oldValues }));
    const service = new GoogleSheetsHostedService({ runtime, fetchFn, now: () => now });
    const preview = await service.updateValues(connectionKey, { spreadsheetId, range, expectedOldValues: oldValues, newValues });
    await expect(service.updateValues(connectionKey, {
      spreadsheetId, range, expectedOldValues: oldValues, newValues,
      previewId: preview.previewId, executionId: preview.executionId,
      approvalDigest: 'f'.repeat(64), confirmed: true,
    })).rejects.toMatchObject({ code: 'CONFIRMATION_REQUIRED' });
    expect(runtime.invoke.mock.calls.some(([, action]) => action === 'claimWrite')).toBe(false);
    expect(fetchFn.mock.calls.every(([, init]) => init?.method !== 'PUT')).toBe(true);
  });

  it('finalizes unknown provider outcomes and never retries a dispatched write', async () => {
    const oldValues = [['old', 'x'], ['y', 1]], newValues = [['new', 'x'], ['y', 2]];
    const runtime = makeRuntime(async (_key, action, request) => {
      if (action === 'claimWrite') return validCredential(request, true);
      if (action === 'finalizeWrite') return { ...request, finalized: true, operation: 'update_values' };
      return validCredential(request);
    });
    let puts = 0;
    const fetchFn = jest.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === 'PUT') { puts++; throw new Error('socket closed'); }
      return response({ values: oldValues });
    });
    const service = new GoogleSheetsHostedService({ runtime, fetchFn, now: () => now });
    const preview = await service.updateValues(connectionKey, { spreadsheetId, range, expectedOldValues: oldValues, newValues });
    const result = await service.updateValues(connectionKey, {
      spreadsheetId, range, expectedOldValues: oldValues, newValues,
      previewId: preview.previewId, executionId: preview.executionId,
      approvalDigest: preview.approvalDigest, confirmed: true,
    });
    expect(result.status).toBe('UNKNOWN_OUTCOME');
    expect(puts).toBe(1);
    expect(runtime.invoke.mock.calls.at(-1)?.[2]).toMatchObject({ status: 'UNKNOWN_OUTCOME' });
  });
});
