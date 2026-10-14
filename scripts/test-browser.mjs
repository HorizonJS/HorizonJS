import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createServer as httpServer } from 'node:http';
import { mkdtemp, readFile, writeFile, rm, realpath } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { chromium, firefox } from 'playwright';
import { build as viteBuild, createServer as viteServer, preview } from 'vite';
import { runSecurityChecks } from './security-checks.mjs';
import { runSafeTemplateChecks } from './safe-template-checks.mjs';

const root = fileURLToPath(new URL('../', import.meta.url));
const temporary = await realpath(await mkdtemp(path.join(tmpdir(), 'horizon-browser-')));
const npmCLI = [process.env.npm_execpath,
  path.join(path.dirname(process.execPath), 'node_modules/npm/bin/npm-cli.js'),
  path.resolve(path.dirname(process.execPath), '../lib/node_modules/npm/bin/npm-cli.js')
].find(candidate => candidate && existsSync(candidate));
if (!npmCLI) throw new Error('Run this check through npm run test:browser so npm_execpath is available.');
function npmRun(args) {
  return execFileSync(process.execPath, [npmCLI, ...args], { cwd: temporary, encoding: 'utf8' });
}

const checks = `
${runSecurityChecks.toString()}
runSecurityChecks(Ext);
${runSafeTemplateChecks.toString()}
runSafeTemplateChecks(Ext);
function check(ok, message) { if (!ok) throw new Error(message); }
check(Ext === window.Ext, 'module and global Ext differ');
await new Promise(resolve => Ext.onReady(resolve));
Ext.namespace('BrowserSmoke.widgets');
check(window.BrowserSmoke.widgets, 'namespace creation');
const basic = new Ext.Template('<b>{name:htmlEncode}</b>').compile();
check(basic.apply({name: '<hello>'}) === '<b>&lt;hello&gt;</b>', 'compiled template');
const advanced = new Ext.XTemplate('<tpl for="."><span>{name}</span></tpl>');
check(advanced.apply([{name:'one'},{name:'two'}]) === '<span>one</span><span>two</span>', 'XTemplate');
check(new Date(2026, 9, 12).format('Y-m-d') === '2026-10-12', 'date format');
check(Date.parseDate('2026-10-12', 'Y-m-d').getDate() === 12, 'date parsing');
const store = new Ext.data.JsonStore({fields:['name'], data:[{name:'Ada'},{name:'Grace'}]});
check(store.getCount() === 2, 'store loading');
const panel = new Ext.Panel({renderTo:'app', title:'HorizonJS', width:360, height:140, html:'<div class="smoke-row">Ready</div>'});
const grid = new Ext.grid.GridPanel({renderTo:'grid', store, width:360, height:160, columns:[{header:'Name', dataIndex:'name'}]});
const form = new Ext.form.FormPanel({renderTo:'form', width:360, items:[{xtype:'textfield', name:'name', fieldLabel:'Name', value:'Ada'}, {xtype:'datefield', name:'date', value:'10/12/2026'}]});
check(form.getForm().getValues().name === 'Ada', 'form fields');
const dateField = form.getForm().findField('date');
check(dateField.getValue().getDate() === 12, 'date field');
dateField.onTriggerClick();
check(dateField.menu.isVisible(), 'date picker popup');
dateField.menu.hide();
check(Ext.query('#app .smoke-row')[0].textContent === 'Ready', 'selector compilation');
check(Ext.query('#grid .x-grid3-row').length === 2, 'grid rendering');
const button = new Ext.Button({renderTo:'button', text:'Click', handler:function(){window.clicked = true;}});
button.el.dom.querySelector('button').click();
check(window.clicked, 'native event handling');
const event = new MouseEvent('click');
check(Ext.lib.Event.getEvent(event) === event, 'explicit event access');
check(Ext.lib.Event.getEvent() === window.event, 'event access without caller inspection');
check(Ext.get('app').getStyle('position') === 'static', 'element style');
check(getComputedStyle(panel.body.dom).overflow === 'hidden', 'package stylesheet');
const fx = Ext.get('effect');
const drag = new Ext.dd.DD('effect', 'module-smoke');
const target = new Ext.dd.DDTarget('button', 'module-smoke');
check(Ext.dd.DragDropMgr.getRelated(drag, true).indexOf(target) !== -1, 'drag/drop group lookup');
drag.unreg(); target.unreg();
// The old standalone splitter is exported alongside the current GridView
// splitter. Exercise its strict-mode event path without starting a drag.
const splitter = new Ext.grid.SplitDragZone(grid, grid.getView().mainHd.dom, grid.getView().mainBody.dom);
splitter.handleMouseDown({target: document.getElementById('app'), type:'mousedown'});
splitter.unreg();
await new Promise(resolve => fx.fadeOut({duration:0.01, callback:resolve}));
await new Promise(resolve => fx.fadeIn({duration:0.01, callback:resolve}));
await new Promise(resolve => fx.slideOut('t', {duration:0.01, callback:resolve}));
await new Promise(resolve => fx.slideIn('t', {duration:0.01, callback:resolve}));
fx.fadeOut({duration:2}); fx.stopFx();
check(!fx.hasActiveFx(), 'stopping queued animation');
await new Promise(resolve => fx.highlight('ffff00', {duration:0.01, callback:resolve}));
await new Promise(resolve => fx.scale(50, 50, {duration:0.01, callback:resolve}));
await new Promise(resolve => fx.shift({x:10, y:10, duration:0.01, callback:resolve}));
await new Promise(resolve => fx.puff({duration:0.01, callback:resolve}));
fx.show();
await new Promise(resolve => fx.ghost('t', {duration:0.01, callback:resolve}));
const blank = await fetch(Ext.BLANK_IMAGE_URL);
check(blank.ok && (await blank.blob()).size > 0, 'blank image URL');
window.smokeDone = true;
`;

const body = '<div id="app"></div><div id="grid"></div><div id="form"></div><div id="button"></div><div id="effect" style="width:40px;height:40px">Effect</div>';
const pageHTML = (head, script) => `<!doctype html><html><head>${head}</head><body>${body}<script type="module">${(script + '\n' + checks).replace(/<\/script/gi, '<\\/script')}</script></body></html>`;

async function staticServer(directory) {
  const server = httpServer(async (request, response) => {
    try {
      const relative = decodeURIComponent(new URL(request.url, 'http://localhost').pathname);
      const filename = path.resolve(directory, '.' + (relative === '/' ? '/index.html' : relative));
      if (!filename.startsWith(path.resolve(directory) + path.sep)) throw new Error('invalid path');
      const types = {'.html':'text/html', '.js':'text/javascript', '.mjs':'text/javascript', '.css':'text/css', '.gif':'image/gif', '.png':'image/png'};
      response.setHeader('Content-Type', types[path.extname(filename)] || 'application/octet-stream');
      response.end(await readFile(filename));
    } catch {
      response.statusCode = 404;
      response.end('Not found');
    }
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  return {url: `http://127.0.0.1:${server.address().port}`, close: () => new Promise(resolve => {server.close(resolve); server.closeAllConnections();})};
}

let browser;
const servers = [];
try {
  // Verify the tarball, not a symlink to the working tree.
  const packed = JSON.parse(npmRun(['pack', path.join(root, 'build/npm'), '--json']));
  const names = packed[0].files.map(file => file.path);
  for (const file of ['index.mjs', 'LICENSE.txt', 'resources/css/ext-all.css', 'resources/images/default/s.gif']) {
    assert(names.includes(file), `tarball missing ${file}`);
  }
  assert(!names.some(name => name.startsWith('docs/') || name.startsWith('examples/')));
  await writeFile(path.join(temporary, 'package.json'), '{"private":true,"type":"module"}');
  npmRun(['install', packed[0].filename, '--ignore-scripts', '--no-audit', '--no-fund']);
  const installed = path.join(temporary, 'node_modules/@horizonjs/core');
  await assert.rejects(import(pathToFileURL(path.join(installed, 'index.mjs'))), /requires a browser DOM/);

  // Check every packaged CSS reference even if the smoke widgets do not use it.
  for (const cssName of ['ext-all.css', 'xtheme-gray.css']) {
    const cssPath = path.join(installed, 'resources/css', cssName);
    const css = await readFile(cssPath, 'utf8');
    for (const match of css.matchAll(/url\(\s*['"]?([^)'"\s]+)['"]?\s*\)/g)) {
      if (!/^(data:|https?:)/.test(match[1])) await readFile(path.resolve(path.dirname(cssPath), match[1]));
    }
  }
  await writeFile(path.join(installed, 'native.html'), pageHTML('<link rel="stylesheet" href="./resources/css/ext-all.css">', `
import Ext, {Ext as named} from './index.mjs';
if(Ext !== named || (await import('./index.mjs')).default !== Ext) throw new Error('module singleton');
try {
  await import('./index.mjs?duplicate');
  throw new Error('duplicate distribution was accepted');
} catch(error) {
  if(!error.message.includes('already loaded')) throw error;
}
`));
  await writeFile(path.join(installed, 'late.html'), pageHTML('<link rel="stylesheet" href="./resources/css/ext-all.css">', "await new Promise(resolve => window.addEventListener('load', resolve)); const {default:Ext} = await import('./index.mjs');"));
  await writeFile(path.join(installed, 'classic.html'), pageHTML('<link rel="stylesheet" href="./resources/css/ext-all.css"><script src="./adapter/ext/ext-base.js"></script><script src="./ext-all.js"></script>', "const Ext = window.Ext; Ext.BLANK_IMAGE_URL = './resources/images/default/s.gif';"));
  await writeFile(path.join(installed, 'classic-debug.html'), pageHTML('<link rel="stylesheet" href="./resources/css/ext-all.css"><script src="./adapter/ext/ext-base.js"></script><script src="./ext-all-debug.js"></script>', "const Ext = window.Ext; Ext.BLANK_IMAGE_URL = './resources/images/default/s.gif';"));
  const native = await staticServer(installed);
  servers.push(native);
  browser = await (process.argv.includes('--firefox') ? firefox : chromium).launch({headless:true});

  async function runPage(label, url) {
    const page = await browser.newPage();
    const errors = [];
    page.on('pageerror', error => errors.push(error.stack || error.message || String(error)));
    page.on('response', response => { if (response.status() >= 400 && !response.url().endsWith('favicon.ico')) errors.push(`${response.status()} ${response.url()}`); });
    await page.goto(url);
    try {
      await page.waitForFunction(() => window.smokeDone, {timeout:15000});
      assert.deepEqual(errors, [], label);
    } catch (error) {
      throw new Error(`${label}: ${errors.join('; ') || error.message}`);
    } finally {
      await page.close();
    }
    console.log(`Passed: ${label}`);
  }
  await runPage('native module', native.url + '/native.html');
  await runPage('module imported after load', native.url + '/late.html');
  await runPage('classic scripts', native.url + '/classic.html');
  await runPage('classic debug scripts', native.url + '/classic-debug.html');

  await writeFile(path.join(temporary, 'index.html'), pageHTML('', "import Ext from '@horizonjs/core'; import '@horizonjs/core/styles.css'; import '@horizonjs/core/themes/gray.css';"));
  const dev = await viteServer({root:temporary, configFile:false, server:{host:'127.0.0.1', port:0}, logLevel:'warn'});
  await dev.listen();
  servers.push({close: () => dev.close()});
  await runPage('Vite development', dev.resolvedUrls.local[0]);
  await viteBuild({root:temporary, configFile:false, logLevel:'warn', base:'/demo/', build:{outDir:'dist', assetsInlineLimit:0}});
  const production = await preview({root:temporary, configFile:false, base:'/demo/', preview:{host:'127.0.0.1', port:0}, logLevel:'warn'});
  servers.push({close: () => new Promise(resolve => production.httpServer.close(resolve))});
  await runPage('Vite production under /demo/', production.resolvedUrls.local[0]);
  console.log('Package tarball and CSS asset references verified.');
} finally {
  if (browser) await browser.close();
  for (const server of servers.reverse()) await server.close();
  // Only remove the exact temporary directory created above.
  await rm(temporary, {recursive:true, force:true});
}
