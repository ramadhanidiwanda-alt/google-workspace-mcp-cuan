/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { CuanSheetsRuntimeClient } from './hostedSheets/CuanSheetsRuntimeClient';
import { GoogleSheetsHostedService } from './hostedSheets/GoogleSheetsHostedService';
import { startHostedHttpServer } from './hostedSheets/httpServer';

if (process.env.CUAN_SHEETS_MCP_ENABLED !== 'true') {
  console.error('Hosted Google Sheets MCP is disabled; set CUAN_SHEETS_MCP_ENABLED=true explicitly.');
  process.exit(1);
}
const runtimeUrl=process.env.CUAN_SHEETS_RUNTIME_URL ?? '';
const runtime=new CuanSheetsRuntimeClient(runtimeUrl,process.env.GOOGLE_SHEETS_PRIVATE_SERVICE_ID ?? '',process.env.GOOGLE_SHEETS_PRIVATE_SERVICE_SECRET ?? '');
const service=new GoogleSheetsHostedService({runtime});
const port=Number(process.env.PORT ?? '8080');
if(!Number.isInteger(port)||port<1||port>65535) throw new Error('Invalid PORT');
startHostedHttpServer(service,process.env.HOST ?? '0.0.0.0',port);
