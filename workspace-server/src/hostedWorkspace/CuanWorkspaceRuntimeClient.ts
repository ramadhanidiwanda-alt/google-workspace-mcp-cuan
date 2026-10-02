/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

export type WorkspaceAction = 'resolveRead' | 'claimWrite' | 'finalizeWrite';
export type WorkspaceRuntime = {
  invoke(key: string, action: WorkspaceAction, request: Record<string, unknown>): Promise<Record<string, unknown>>;
  redeem?(googleInvocation: Record<string, unknown>): Promise<Record<string, unknown>>;
  finalize?(googleInvocation: Record<string, unknown>, outcome: 'succeeded' | 'failed_before_dispatch' | 'failed_after_dispatch'): Promise<void>;
};

export class CuanWorkspaceRuntimeClient implements WorkspaceRuntime {
  constructor(
    private readonly url: string,
    private readonly serviceId: string,
    private readonly serviceSecret: string,
    private readonly fetchFn: typeof fetch = fetch,
    private readonly redeemUrl: string = process.env.CUAN_GOOGLE_WORKSPACE_REDEEM_URL ?? new URL('mcp-redeem-google-permit', url).toString(),
  ) {
    const parsed = new URL(url);
    if (parsed.protocol !== 'https:' || parsed.search || parsed.hash || parsed.username || parsed.password) throw new Error('Invalid Cuan Workspace runtime URL');
    if (!serviceId || !serviceSecret) throw new Error('Cuan Workspace service credentials are required');
  }

  async redeem(googleInvocation: Record<string, unknown>): Promise<Record<string, unknown>> {
    if (!this.redeemUrl) throw new Error('CUAN_WORKSPACE_REDEEM_UNAVAILABLE');
    const parsedUrl = new URL(this.redeemUrl);
    if (parsedUrl.protocol !== 'https:' || parsedUrl.search || parsedUrl.hash || parsedUrl.username || parsedUrl.password) throw new Error('CUAN_WORKSPACE_REDEEM_URL_INVALID');
    const response = await this.fetchFn(this.redeemUrl, {
      method: 'POST', redirect: 'error', signal: AbortSignal.timeout(8000),
      headers: { 'content-type': 'application/json',
        'x-cuan-google-workspace-service-id': this.serviceId,
        'x-cuan-google-workspace-service-secret': this.serviceSecret },
      body: JSON.stringify({ googleInvocation }),
    });
    if (!response.ok) throw new Error('CUAN_WORKSPACE_REDEEM_DENIED');
    const raw = await response.text();
    if (Buffer.byteLength(raw) > 16384) throw new Error('CUAN_WORKSPACE_REDEEM_INVALID');
    let parsed: unknown;
    try { parsed = JSON.parse(raw); } catch { throw new Error('CUAN_WORKSPACE_REDEEM_INVALID'); }
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed) || (parsed as Record<string, unknown>).ok !== true) throw new Error('CUAN_WORKSPACE_REDEEM_DENIED');
    return parsed as Record<string, unknown>;
  }

  async finalize(googleInvocation: Record<string, unknown>, outcome: 'succeeded' | 'failed_before_dispatch' | 'failed_after_dispatch'): Promise<void> {
    const url = new URL('mcp-finalize-execution', this.redeemUrl).toString();
    const response = await this.fetchFn(url, {
      method: 'POST', redirect: 'error', signal: AbortSignal.timeout(8000),
      headers: { 'content-type': 'application/json',
        'x-cuan-google-workspace-service-id': this.serviceId,
        'x-cuan-google-workspace-service-secret': this.serviceSecret },
      body: JSON.stringify({ executionId: googleInvocation.executionId, permit: googleInvocation.permit, outcome }),
    });
    if (!response.ok) throw new Error('CUAN_WORKSPACE_FINALIZATION_UNKNOWN');
    const raw = await response.text();
    if (Buffer.byteLength(raw) > 8192) throw new Error('CUAN_WORKSPACE_FINALIZATION_UNKNOWN');
    let parsed: unknown;
    try { parsed = JSON.parse(raw); } catch { throw new Error('CUAN_WORKSPACE_FINALIZATION_UNKNOWN'); }
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed) || (parsed as Record<string, unknown>).ok !== true)
      throw new Error('CUAN_WORKSPACE_FINALIZATION_UNKNOWN');
  }

  async invoke(key: string, action: WorkspaceAction, request: Record<string, unknown>): Promise<Record<string, unknown>> {
    const response = await this.fetchFn(this.url, {
      method: 'POST', redirect: 'error', signal: AbortSignal.timeout(8000),
      headers: {
        'content-type': 'application/json',
        'x-cuan-google-workspace-service-id': this.serviceId,
        'x-cuan-google-workspace-service-secret': this.serviceSecret,
        'x-cuan-mcp-connection-key': key,
      },
      body: JSON.stringify({ action, request }),
    });
    if (!response.ok) throw new Error('CUAN_WORKSPACE_RUNTIME_REJECTED');
    if (!response.body) throw new Error('CUAN_WORKSPACE_RUNTIME_INVALID_RESPONSE');
    const reader = response.body.getReader();
    const chunks: Uint8Array[] = []; let size = 0;
    while (true) {
      const part = await reader.read();
      if (part.done) break;
      size += part.value.byteLength;
      if (size > 8192) { await reader.cancel(); throw new Error('CUAN_WORKSPACE_RUNTIME_INVALID_RESPONSE'); }
      chunks.push(part.value);
    }
    const body = Buffer.concat(chunks.map(chunk => Buffer.from(chunk))).toString('utf8');
    let parsed: unknown;
    try { parsed = JSON.parse(body); } catch { throw new Error('CUAN_WORKSPACE_RUNTIME_INVALID_RESPONSE'); }
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error('CUAN_WORKSPACE_RUNTIME_INVALID_RESPONSE');
    return parsed as Record<string, unknown>;
  }
}
