/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

import { CuanWorkspaceRuntimeClient } from './hostedWorkspace/CuanWorkspaceRuntimeClient';
import { GoogleWorkspaceHostedService } from './hostedWorkspace/GoogleWorkspaceHostedService';
import { startHostedWorkspaceHttpServer } from './hostedWorkspace/httpServer';

if (process.env.CUAN_WORKSPACE_MCP_ENABLED !== 'true') {
  console.error('Hosted Google Workspace MCP is disabled; set CUAN_WORKSPACE_MCP_ENABLED=true explicitly.');
  process.exit(1);
}
const runtime = new CuanWorkspaceRuntimeClient(
  process.env.CUAN_WORKSPACE_RUNTIME_URL ?? '',
  process.env.GOOGLE_WORKSPACE_PRIVATE_SERVICE_ID ?? '',
  process.env.GOOGLE_WORKSPACE_PRIVATE_SERVICE_SECRET ?? '',
);
const service = new GoogleWorkspaceHostedService({ runtime });
const port = Number(process.env.PORT ?? '8080');
if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error('Invalid PORT');
startHostedWorkspaceHttpServer(service, process.env.HOST ?? '0.0.0.0', port);
