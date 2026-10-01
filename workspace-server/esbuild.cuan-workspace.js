/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

const esbuild = require('esbuild');
esbuild.build({ entryPoints: ['src/cuanWorkspaceHost.ts'], bundle: true, platform: 'node', target: 'node20', outfile: 'dist/cuan-workspace.js', format: 'cjs', minify: true, external: [], logLevel: 'info' }).catch(error => { console.error(error); process.exit(1); });
