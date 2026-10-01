/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { createHash, randomUUID } from 'node:crypto';
import type { WorkspaceRuntime } from './CuanWorkspaceRuntimeClient';

type Cell = string | number | boolean;
type Grid = Cell[][];
type Binding = { version: 1; toolName: string; targetId: string; executionId: string; requestDigest: string };
type Credential = { accessToken: string; accountId: string; credentialVersion: number };
type Confirmation = { previewId?: string; approvalDigest?: string; confirmed?: boolean };
export class HostedWorkspaceError extends Error { constructor(public readonly code: string) { super(code); } }

const KEY = /^ci_mcp_ck_[0-9a-f]{64}$/;
const ID = /^[A-Za-z0-9_-]{10,200}$/;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const RANGE = /^'([A-Za-z0-9 _-]{1,60})'!([A-Z]{1,3})([1-9]\d{0,4}):([A-Z]{1,3})([1-9]\d{0,4})$/;
const TOOLS = {
  list: 'workspace_drive_list_files', file: 'workspace_drive_get_file', text: 'workspace_drive_get_text',
  sheetRead: 'workspace_sheets_read_values', create: 'workspace_drive_create_text_file',
  update: 'workspace_drive_update_text_file', sheetWrite: 'workspace_sheets_update_values',
} as const;
const sha = (value: unknown) => createHash('sha256').update(JSON.stringify(value)).digest('hex');
const contentDigest = (value: string) => createHash('sha256').update(value).digest('hex');
const column = (value: string) => [...value].reduce((n, ch) => n * 26 + ch.charCodeAt(0) - 64, 0);
function fileId(value: unknown): string {
  if (typeof value !== 'string' || !ID.test(value)) throw new HostedWorkspaceError('INVALID_TARGET');
  return value;
}
function text(value: unknown, max = 65536): string {
  if (typeof value !== 'string' || value.length > max || /[\u0000-\u0008\u000b\u000c\u000e-\u001f]/.test(value)) throw new HostedWorkspaceError('INVALID_TEXT');
  return value;
}
function shape(range: unknown) {
  if (typeof range !== 'string' || range.length > 128) throw new HostedWorkspaceError('INVALID_RANGE');
  const match = RANGE.exec(range);
  if (!match) throw new HostedWorkspaceError('INVALID_RANGE');
  const height = Number(match[5]) - Number(match[3]) + 1;
  const width = column(match[4]) - column(match[2]) + 1;
  if (height < 1 || width < 1 || height * width > 100) throw new HostedWorkspaceError('INVALID_RANGE');
  return { height, width };
}
function grid(value: unknown, dims: { height: number; width: number }, rectangular = false): Grid {
  if (!Array.isArray(value) || value.length > dims.height || value.some(row => !Array.isArray(row) || row.length > dims.width || row.some((v: unknown) =>
    !(typeof v === 'string' && v.length <= 256 && !/[\u0000-\u0008\u000b\u000c\u000e-\u001f]/.test(v) || typeof v === 'number' && Number.isFinite(v) || typeof v === 'boolean')))) throw new HostedWorkspaceError('INVALID_VALUES');
  if (rectangular && (value.length !== dims.height || value.some(row => row.length !== dims.width))) throw new HostedWorkspaceError('INVALID_VALUES');
  return Array.from({ length: dims.height }, (_, r) => Array.from({ length: dims.width }, (_, c) => value[r]?.[c] ?? '')) as Grid;
}
function exactCredential(raw: Record<string, unknown>, binding: Binding, now: number, previous?: Credential, claimed = false): Credential {
  if (raw.version !== 1 || raw.toolName !== binding.toolName || raw.targetId !== binding.targetId || raw.executionId !== binding.executionId || raw.requestDigest !== binding.requestDigest ||
    typeof raw.accountId !== 'string' || !ID.test(raw.accountId) || typeof raw.credentialVersion !== 'number' || !Number.isSafeInteger(raw.credentialVersion) || raw.credentialVersion < 1 ||
    typeof raw.expiresAt !== 'number' || raw.expiresAt <= now || raw.expiresAt > now + 10 * 60_000 ||
    typeof raw.accessToken !== 'string' || raw.accessToken.length < 8 || raw.accessToken.length > 4096 ||
    previous && (raw.accountId !== previous.accountId || raw.credentialVersion !== previous.credentialVersion) ||
    claimed && raw.writeClaimed !== true) throw new HostedWorkspaceError('CUAN_BINDING_MISMATCH');
  return { accessToken: raw.accessToken as string, accountId: raw.accountId as string, credentialVersion: raw.credentialVersion as number };
}

export class GoogleWorkspaceHostedService {
  constructor(private readonly deps: { runtime: WorkspaceRuntime; fetchFn?: typeof fetch; now?: () => number }) {}
  private now() { return (this.deps.now ?? Date.now)(); }
  private checkKey(key: string) { if (!KEY.test(key)) throw new HostedWorkspaceError('UNAUTHENTICATED'); }
  private binding(toolName: string, targetId: string, input: unknown, executionId = randomUUID()): Binding {
    return { version: 1, toolName, targetId, executionId, requestDigest: sha([toolName, targetId, input]) };
  }
  private async authorize(key: string, action: 'resolveRead' | 'claimWrite', binding: Binding, extra: Record<string, unknown> = {}, previous?: Credential): Promise<Credential> {
    const result = await this.deps.runtime.invoke(key, action, { ...binding, ...extra });
    return exactCredential(result, binding, this.now(), previous, action === 'claimWrite');
  }
  private async provider(token: string, url: string, method: 'GET' | 'POST' | 'PATCH' | 'PUT' = 'GET', body?: string, contentType?: string, maxBytes = 131072): Promise<{ json?: Record<string, unknown>; text?: string }> {
    let response: Response;
    try {
      response = await (this.deps.fetchFn ?? fetch)(url, { method, redirect: 'error', signal: AbortSignal.timeout(7000), headers: { authorization: `Bearer ${token}`, ...(contentType ? { 'content-type': contentType } : {}) }, ...(body === undefined ? {} : { body }) });
    } catch { throw new HostedWorkspaceError(method === 'GET' ? 'PROVIDER_READ_FAILED' : 'PROVIDER_OUTCOME_UNKNOWN'); }
    if (!response.ok) throw new HostedWorkspaceError(method === 'GET' ? 'PROVIDER_READ_FAILED' : 'PROVIDER_OUTCOME_UNKNOWN');
    const contentLength = Number(response.headers.get('content-length') ?? 0);
    if (contentLength > maxBytes) throw new HostedWorkspaceError(method === 'GET' ? 'PROVIDER_RESPONSE_TOO_LARGE' : 'PROVIDER_OUTCOME_UNKNOWN');
    let raw: string;
    try {
      if (!response.body) throw new Error('empty response');
      const reader = response.body.getReader();
      const chunks: Uint8Array[] = []; let size = 0;
      while (true) {
        const part = await reader.read();
        if (part.done) break;
        size += part.value.byteLength;
        if (size > maxBytes) { await reader.cancel(); throw new HostedWorkspaceError(method === 'GET' ? 'PROVIDER_RESPONSE_TOO_LARGE' : 'PROVIDER_OUTCOME_UNKNOWN'); }
        chunks.push(part.value);
      }
      raw = Buffer.concat(chunks.map(chunk => Buffer.from(chunk))).toString('utf8');
    } catch (error) {
      if (error instanceof HostedWorkspaceError) throw error;
      throw new HostedWorkspaceError(method === 'GET' ? 'PROVIDER_READ_FAILED' : 'PROVIDER_OUTCOME_UNKNOWN');
    }
    if (response.headers.get('content-type')?.includes('application/json')) {
      try {
        const parsed: unknown = JSON.parse(raw);
        if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error();
        return { json: parsed as Record<string, unknown> };
      } catch { throw new HostedWorkspaceError(method === 'GET' ? 'PROVIDER_READ_FAILED' : 'PROVIDER_OUTCOME_UNKNOWN'); }
    }
    return { text: raw };
  }
  private async metadata(token: string, id: string) {
    const url = `https://www.googleapis.com/drive/v3/files/${id}?fields=id,name,mimeType,size,modifiedTime,webViewLink,description,parents,trashed&supportsAllDrives=true`;
    const raw = (await this.provider(token, url)).json;
    if (!raw || raw.id !== id || raw.trashed === true || typeof raw.mimeType !== 'string') throw new HostedWorkspaceError('PROVIDER_INVALID_FILE');
    return raw;
  }
  private async fileText(token: string, id: string) {
    const meta = await this.metadata(token, id);
    let url: string;
    if (meta.mimeType === 'application/vnd.google-apps.document') url = `https://www.googleapis.com/drive/v3/files/${id}/export?mimeType=text%2Fplain`;
    else if (meta.mimeType === 'text/plain') url = `https://www.googleapis.com/drive/v3/files/${id}?alt=media&supportsAllDrives=true`;
    else throw new HostedWorkspaceError('UNSUPPORTED_MIME_TYPE');
    const value = await this.provider(token, url, 'GET', undefined, undefined, 65536);
    return { metadata: meta, content: text(value.text ?? '', 65536) };
  }
  async listFiles(key: string, input: { pageSize?: number; pageToken?: string } = {}) {
    this.checkKey(key);
    const pageSize = input.pageSize ?? 50;
    if (!Number.isInteger(pageSize) || pageSize < 1 || pageSize > 100 || input.pageToken !== undefined && (typeof input.pageToken !== 'string' || input.pageToken.length > 1000 || !/^[A-Za-z0-9_+/=-]+$/.test(input.pageToken))) throw new HostedWorkspaceError('INVALID_PAGE');
    const binding = this.binding(TOOLS.list, 'account-root', { pageSize, pageToken: input.pageToken ?? '' });
    const credential = await this.authorize(key, 'resolveRead', binding);
    const query = new URLSearchParams({ pageSize: String(pageSize), q: 'trashed = false', fields: 'nextPageToken,files(id,name,mimeType,size,modifiedTime,webViewLink,description,parents)', supportsAllDrives: 'true', includeItemsFromAllDrives: 'true' });
    if (input.pageToken) query.set('pageToken', input.pageToken);
    const raw = (await this.provider(credential.accessToken, `https://www.googleapis.com/drive/v3/files?${query}`)).json;
    if (!raw || !Array.isArray(raw.files) || raw.files.length > pageSize) throw new HostedWorkspaceError('PROVIDER_INVALID_LIST');
    return { files: raw.files, ...(typeof raw.nextPageToken === 'string' && raw.nextPageToken.length <= 1000 ? { nextPageToken: raw.nextPageToken } : {}) };
  }
  async getFile(key: string, input: { fileId: string }) {
    this.checkKey(key); const id = fileId(input.fileId);
    const binding = this.binding(TOOLS.file, id, null);
    const credential = await this.authorize(key, 'resolveRead', binding);
    return { file: await this.metadata(credential.accessToken, id) };
  }
  async getText(key: string, input: { fileId: string }) {
    this.checkKey(key); const id = fileId(input.fileId);
    const binding = this.binding(TOOLS.text, id, null);
    const credential = await this.authorize(key, 'resolveRead', binding);
    const result = await this.fileText(credential.accessToken, id);
    return { fileId: id, name: result.metadata.name, mimeType: result.metadata.mimeType, content: result.content };
  }
  async readValues(key: string, input: { spreadsheetId: string; range: string }) {
    this.checkKey(key); const id = fileId(input.spreadsheetId), dims = shape(input.range);
    const binding = this.binding(TOOLS.sheetRead, id, input.range);
    const credential = await this.authorize(key, 'resolveRead', binding);
    const url = `https://sheets.googleapis.com/v4/spreadsheets/${id}/values/${encodeURIComponent(input.range)}?majorDimension=ROWS&valueRenderOption=FORMULA`;
    const raw = (await this.provider(credential.accessToken, url)).json;
    if (!raw) throw new HostedWorkspaceError('PROVIDER_INVALID_VALUES');
    return { spreadsheetId: id, range: input.range, values: grid(raw.values ?? [], dims) };
  }
  private confirmation(input: Confirmation, digest: string) {
    if (!UUID.test(input.previewId ?? '') || input.approvalDigest !== digest || !input.confirmed) throw new HostedWorkspaceError('CONFIRMATION_REQUIRED');
  }
  private async finalize(key: string, binding: Binding, previewId: string, status: 'CONFIRMED' | 'UNKNOWN_OUTCOME', previous: Credential) {
    const raw = await this.deps.runtime.invoke(key, 'finalizeWrite', { ...binding, previewId, status });
    if (raw.version !== 1 || raw.finalized !== true || raw.status !== status || raw.accountId !== previous.accountId || raw.credentialVersion !== previous.credentialVersion ||
      typeof raw.expiresAt !== 'number' || !Number.isFinite(raw.expiresAt) || raw.expiresAt > this.now() + 10 * 60_000 ||
      raw.toolName !== binding.toolName || raw.targetId !== binding.targetId || raw.executionId !== binding.executionId || raw.requestDigest !== binding.requestDigest) throw new HostedWorkspaceError('CUAN_FINALIZE_MISMATCH');
  }
  private async write(key: string, toolName: string, targetId: string, payload: unknown, confirmation: Confirmation, previewRead: (token: string) => Promise<unknown>, dispatch: (token: string) => Promise<unknown>, verify: (token: string, result: unknown) => Promise<void>) {
    this.checkKey(key);
    const digest = sha([toolName, targetId, payload]);
    if (!confirmation.confirmed) {
      const previewId = randomUUID();
      const readBinding: Binding = { version: 1, toolName, targetId, executionId: randomUUID(), requestDigest: digest };
      const readCredential = await this.authorize(key, 'resolveRead', readBinding);
      const before = await previewRead(readCredential.accessToken);
      const details = payload as { name?: string; parentId?: string; range?: string; content?: string; replacement?: string | Grid };
      const beforeContent = toolName === TOOLS.create ? '' : typeof before === 'string' ? before : JSON.stringify(before);
      const afterContent = typeof details.content === 'string' ? details.content : typeof details.replacement === 'string' ? details.replacement : JSON.stringify(details.replacement);
      const previewSummary = {
        operation: toolName, targetId,
        ...(details.name ? { name: details.name } : {}),
        ...(details.parentId ? { parentId: details.parentId } : {}),
        ...(details.range ? { range: details.range } : {}),
        beforeExcerpt: beforeContent.slice(0, 512), afterExcerpt: afterContent.slice(0, 512),
        beforeDigest: contentDigest(beforeContent), afterDigest: contentDigest(afterContent),
        beforeLength: beforeContent.length, afterLength: afterContent.length,
      };
      const binding: Binding = { version: 1, toolName, targetId, executionId: previewId, requestDigest: digest };
      await this.authorize(key, 'resolveRead', binding, { preview: true, previewSummary }, readCredential);
      return { targetId, previewId, approvalDigest: digest, before, proposed: payload, requiresConfirmation: true };
    }
    this.confirmation(confirmation, digest);
    const preflight = this.binding(toolName, targetId, payload);
    const preflightCredential = await this.authorize(key, 'resolveRead', preflight);
    const before = await previewRead(preflightCredential.accessToken);
    const expected = (payload as { expected?: unknown }).expected;
    if (expected !== undefined && sha(before) !== sha(expected)) throw new HostedWorkspaceError('STALE_CONTENT');
    const binding: Binding = { version: 1, toolName, targetId, executionId: randomUUID(), requestDigest: digest };
    const credential = await this.authorize(key, 'claimWrite', binding, { previewId: confirmation.previewId }, preflightCredential);
    let status: 'CONFIRMED' | 'UNKNOWN_OUTCOME' = 'UNKNOWN_OUTCOME';
    let result: unknown;
    try {
      result = await dispatch(credential.accessToken);
      await verify(credential.accessToken, result);
      status = 'CONFIRMED';
    } catch { status = 'UNKNOWN_OUTCOME'; }
    await this.finalize(key, binding, confirmation.previewId!, status, credential);
    return { status, targetId, executionId: binding.executionId, ...(status === 'CONFIRMED' ? { result } : {}) };
  }
  async createTextFile(key: string, input: { name: string; content: string; parentId?: string } & Confirmation) {
    const name = text(input.name, 200).trim(), content = text(input.content);
    if (!name || /[\/\\]/.test(name)) throw new HostedWorkspaceError('INVALID_NAME');
    const parentId = input.parentId === undefined ? undefined : fileId(input.parentId);
    const payload = { name, content, ...(parentId ? { parentId } : {}) };
    return this.write(key, TOOLS.create, 'account-root', payload, input, async () => ({ destination: parentId ?? 'root' }), async token => {
      const boundary = `cuan_${randomUUID().replace(/-/g, '')}`;
      const metadata = JSON.stringify({ name, mimeType: 'text/plain', ...(parentId ? { parents: [parentId] } : {}) });
      const body = `--${boundary}\r\nContent-Type: application/json; charset=UTF-8\r\n\r\n${metadata}\r\n--${boundary}\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n${content}\r\n--${boundary}--`;
      const raw = (await this.provider(token, 'https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&supportsAllDrives=true&fields=id,name,mimeType,webViewLink', 'POST', body, `multipart/related; boundary=${boundary}`)).json;
      if (!raw || typeof raw.id !== 'string' || !ID.test(raw.id)) throw new HostedWorkspaceError('PROVIDER_OUTCOME_UNKNOWN');
      return { fileId: raw.id, name: raw.name, mimeType: raw.mimeType, webViewLink: raw.webViewLink };
    }, async (token, result) => {
      const createdId = (result as { fileId: string }).fileId;
      const current = await this.fileText(token, createdId);
      if (current.metadata.name !== name || current.metadata.mimeType !== 'text/plain' || current.content !== content ||
        parentId && (!Array.isArray(current.metadata.parents) || !current.metadata.parents.includes(parentId))) throw new HostedWorkspaceError('PROVIDER_OUTCOME_UNKNOWN');
    });
  }
  async updateTextFile(key: string, input: { fileId: string; expectedOldText: string; newText: string } & Confirmation) {
    const id = fileId(input.fileId), expected = text(input.expectedOldText), replacement = text(input.newText);
    const payload = { expected, replacement };
    return this.write(key, TOOLS.update, id, payload, input, async token => {
      const current = await this.fileText(token, id);
      if (current.metadata.mimeType !== 'text/plain') throw new HostedWorkspaceError('UNSUPPORTED_MIME_TYPE');
      if (current.content !== expected) throw new HostedWorkspaceError('STALE_CONTENT');
      return current.content;
    }, async token => {
      const raw = (await this.provider(token, `https://www.googleapis.com/upload/drive/v3/files/${id}?uploadType=media&supportsAllDrives=true&fields=id,name,mimeType`, 'PATCH', replacement, 'text/plain; charset=UTF-8')).json;
      if (!raw || raw.id !== id) throw new HostedWorkspaceError('PROVIDER_OUTCOME_UNKNOWN');
      return { fileId: id };
    }, async token => {
      const current = await this.fileText(token, id);
      if (current.metadata.mimeType !== 'text/plain' || current.content !== replacement) throw new HostedWorkspaceError('PROVIDER_OUTCOME_UNKNOWN');
    });
  }
  async updateValues(key: string, input: { spreadsheetId: string; range: string; expectedOldValues: Grid; newValues: Grid } & Confirmation) {
    const id = fileId(input.spreadsheetId), dims = shape(input.range);
    const expected = grid(input.expectedOldValues, dims, true), replacement = grid(input.newValues, dims, true);
    const payload = { range: input.range, expected, replacement };
    const url = `https://sheets.googleapis.com/v4/spreadsheets/${id}/values/${encodeURIComponent(input.range)}`;
    return this.write(key, TOOLS.sheetWrite, id, payload, input, async token => {
      const raw = (await this.provider(token, `${url}?majorDimension=ROWS&valueRenderOption=FORMULA`)).json;
      const current = grid(raw?.values ?? [], dims);
      if (sha(current) !== sha(expected)) throw new HostedWorkspaceError('STALE_CONTENT');
      return current;
    }, async token => {
      const raw = (await this.provider(token, `${url}?valueInputOption=RAW`, 'PUT', JSON.stringify({ majorDimension: 'ROWS', values: replacement }), 'application/json')).json;
      if (!raw || raw.updatedRange !== input.range && raw.updatedCells === undefined) throw new HostedWorkspaceError('PROVIDER_OUTCOME_UNKNOWN');
      return { spreadsheetId: id, range: input.range };
    }, async token => {
      const raw = (await this.provider(token, `${url}?majorDimension=ROWS&valueRenderOption=FORMULA`)).json;
      if (!raw || sha(grid(raw.values ?? [], dims)) !== sha(replacement)) throw new HostedWorkspaceError('PROVIDER_OUTCOME_UNKNOWN');
    });
  }
}
