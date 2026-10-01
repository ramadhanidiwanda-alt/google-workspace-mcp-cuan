/**
 * @license
 * Copyright 2026 Ramadhani Diwanda
 * SPDX-License-Identifier: Apache-2.0
 */

const esbuild=require('esbuild');
esbuild.build({entryPoints:['src/cuanSheetsHost.ts'],bundle:true,platform:'node',target:'node20',outfile:'dist/cuan-sheets.js',format:'cjs',minify:true,external:[],logLevel:'info'}).catch(e=>{console.error(e);process.exit(1);});
