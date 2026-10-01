/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

export type RuntimeAction = 'resolveRead' | 'claimWrite' | 'finalizeWrite';
export type CuanSheetsRuntime = { invoke(key: string, action: RuntimeAction, request: Record<string, unknown>): Promise<Record<string, unknown>> };

export class CuanSheetsRuntimeClient implements CuanSheetsRuntime {
  constructor(private readonly url: string, private readonly serviceId: string, private readonly serviceSecret: string, private readonly fetchFn: typeof fetch = fetch) {
    const parsed = new URL(url);
    if (parsed.protocol !== 'https:' || parsed.search || parsed.hash) throw new Error('Invalid Cuan Sheets runtime URL');
    if (!serviceId || !serviceSecret) throw new Error('Cuan Sheets service credentials are required');
  }

  async invoke(key: string, action: RuntimeAction, request: Record<string, unknown>): Promise<Record<string, unknown>> {
    const response = await this.fetchFn(this.url, {
      method: 'POST', redirect: 'error', signal: AbortSignal.timeout(8000),
      headers: { 'content-type': 'application/json', 'x-cuan-google-sheets-service-id': this.serviceId,
        'x-cuan-google-sheets-service-secret': this.serviceSecret, 'x-cuan-mcp-connection-key': key },
      body: JSON.stringify({ action, request }),
    });
    if (!response.ok) throw new Error('CUAN_RUNTIME_REJECTED');
    const body: unknown = await response.json();
    if (!body || typeof body !== 'object' || Array.isArray(body)) throw new Error('CUAN_RUNTIME_INVALID_RESPONSE');
    return body as Record<string, unknown>;
  }
}
