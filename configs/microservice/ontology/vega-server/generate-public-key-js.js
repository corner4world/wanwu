#!/usr/bin/env node
/*
 * 跨平台脚本：将 data-connection/public_key.pem 转换为 web/public-key.js
 * 输出格式: window.__VEGA_PUBLIC_KEY__ = "<PEM 内容>";
 *
 * 用法:
 *   node configs/vega-server/generate-public-key-js.js
 */
const fs = require('fs');
const path = require('path');

const baseDir = __dirname;
const pemPath = path.join(baseDir, 'data-connection', 'public_key.pem');
const outPath = path.join(baseDir, 'web', 'public-key.js');

if (!fs.existsSync(pemPath)) {
  console.error(`Public key not found: ${pemPath}`);
  console.error('Run ./generate-keys.sh (or follow README "手动生成") first.');
  process.exit(1);
}

if (fs.existsSync(outPath)) {
  console.log(`Existing file detected, skipping generation: ${outPath}`);
  console.log('Delete this file first if you want to regenerate.');
  process.exit(0);
}

const pem = fs.readFileSync(pemPath, 'utf8').replace(/\r\n/g, '\n').trim();
const content = `window.__VEGA_PUBLIC_KEY__ = ${JSON.stringify(pem)};\n`;

fs.mkdirSync(path.dirname(outPath), { recursive: true });
fs.writeFileSync(outPath, content, { encoding: 'utf8' });

console.log(`Wrote ${outPath}`);
