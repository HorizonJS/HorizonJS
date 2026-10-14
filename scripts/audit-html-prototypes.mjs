// Manual diagnostic: reports exposures rather than asserting vulnerable behavior in CI.
import { chromium } from 'playwright';
import { createServer } from 'node:http';
import { readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const sources = new Map();
for (const file of ['index.mjs', 'adapter/ext/ext-base.js', 'ext-all.js', 'ext-all-debug.js']) {
  sources.set('/' + file, await readFile(path.join(root, 'build/npm', file)));
}
const server = createServer((request, response) => {
  const source = sources.get(request.url);
  response.setHeader('Content-Type', source ? 'text/javascript' : 'text/html');
  response.end(source || '<!doctype html><body></body>');
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const browser = await chromium.launch({headless:true});
const report = [];
try {
  for (const distribution of ['module', 'classic', 'classic-debug']) {
    const page = await browser.newPage();
    await page.route('**/*', route => {
      const url = new URL(route.request().url());
      return url.hostname === '127.0.0.1' ? route.continue() : route.abort();
    });
    await page.goto(`http://127.0.0.1:${server.address().port}`);
    if (distribution === 'module') {
      await page.addScriptTag({type:'module', content:'import Ext from "/index.mjs"; window.auditReady = true;'});
      await page.waitForFunction(() => window.auditReady);
    } else {
      await page.addScriptTag({url:'/adapter/ext/ext-base.js'});
      await page.addScriptTag({url:distribution === 'classic' ? '/ext-all.js' : '/ext-all-debug.js'});
    }
    const observations = await page.evaluate(async () => {
      Ext.BLANK_IMAGE_URL = 'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7';
      const observations = [];
      const html = '<img src="data:image/png;base64,invalid" onerror="window.__htmlAudit=1">';
      async function probe(name, run) {
        window.__htmlAudit = 0;
        const host = document.createElement('div');
        document.body.appendChild(host);
        let detail, error = null;
        try {detail = await run(host);} catch(e) {error = String(e);}
        await new Promise(resolve => setTimeout(resolve, 40));
        observations.push({name, executed:window.__htmlAudit === 1, detail, error});
        host.remove();
      }
      await probe('Grid default renderer with strict-JSON store data', host => {
        const store = new Ext.data.JsonStore({fields:['name'], data:JSON.parse(JSON.stringify([{name:html}]))});
        const grid = new Ext.grid.GridPanel({renderTo:host, store, width:240, height:90, columns:[{header:'Name',dataIndex:'name'}]});
        return {insertedImage:!!grid.getEl().dom.querySelector('img[onerror]')};
      });
      await probe('TreeLoader strict-JSON node text', host => {
        const loader = new Ext.tree.TreeLoader();
        const node = loader.createNode(JSON.parse(JSON.stringify({id:'audit-tree',leaf:true,text:html})));
        node.ui.renderElements(node, node.attributes, host, false);
        return {insertedImage:!!host.querySelector('img[onerror]')};
      });
      await probe('Tree node class attribute breakout', host => {
        const node = new Ext.tree.TreeLoader().createNode(JSON.parse(JSON.stringify({id:'audit-class',leaf:true,text:'safe',cls:'" onmouseover="window.__htmlAudit=1'})));
        node.ui.renderElements(node, node.attributes, host, false);
        node.ui.elNode.dispatchEvent(new MouseEvent('mouseover', {bubbles:true}));
        return {injectedHandler:node.ui.elNode.getAttribute('onmouseover')};
      });
      await probe('Tree javascript href navigation', host => {
        const node = new Ext.tree.TreeLoader().createNode({id:'audit-url',leaf:true,text:'safe',href:'javascript:window.__htmlAudit=1;void 0'});
        node.ui.renderElements(node, node.attributes, host, false);
        node.ui.anchor.click();
        return {href:node.ui.anchor.getAttribute('href')};
      });
      await probe('Form validation message from strict JSON', host => {
        const target = document.createElement('div');
        target.id = 'audit-message';
        host.appendChild(target);
        const field = new Ext.form.TextField({renderTo:host, id:'audit-field',msgTarget:target.id});
        const response = Ext.decode(JSON.stringify({success:false,errors:[{id:'audit-field',msg:html}]}));
        Ext.form.BasicForm.prototype.markInvalid.call({findField(){return field;}}, response.errors);
        return {insertedImage:!!target.querySelector('img[onerror]')};
      });
      await probe('DomHelper ordinary attribute breakout', host => {
        const element = Ext.DomHelper.append(host, {tag:'div',title:'" onmouseover="window.__htmlAudit=1'});
        element.dispatchEvent(new MouseEvent('mouseover', {bubbles:true}));
        return {injectedHandler:element.getAttribute('onmouseover')};
      });
      await probe('htmlEncode in a single-quoted template attribute', host => {
        const template = new Ext.Template("<div title='{name:htmlEncode}'>safe</div>");
        const element = template.append(host, {name:"' onmouseover='window.__htmlAudit=1"});
        element.dispatchEvent(new MouseEvent('mouseover', {bubbles:true}));
        return {injectedHandler:element.getAttribute('onmouseover')};
      });
      const target = {};
      Ext.apply(target, Ext.decode('{"__proto__":{"auditInherited":"injected"}}'));
      observations.push({name:'Ext.apply receiver prototype mutation', prototypeChanged:Object.getPrototypeOf(target) !== Object.prototype, inheritedValue:target.auditInherited, globalObjectPrototypeChanged:Object.prototype.hasOwnProperty.call(Object.prototype,'auditInherited')});
      const inherited = Object.create({inheritedOption:'injected'});
      const copied = Ext.apply({}, inherited), defaulted = Ext.applyIf({}, inherited);
      observations.push({name:'Inherited options copied by apply/applyIf', applyCopiedOwn:Object.prototype.hasOwnProperty.call(copied,'inheritedOption'), applyIfCopiedOwn:Object.prototype.hasOwnProperty.call(defaulted,'inheritedOption')});
      const collection = new Ext.util.MixedCollection();
      const before = {containsProto:collection.containsKey('__proto__'), containsConstructor:collection.containsKey('constructor'), protoLookupIsObjectPrototype:collection.key('__proto__') === Object.prototype};
      collection.add('__proto__', {id:'__proto__',phantom:'injected'});
      observations.push({name:'MixedCollection dictionary collisions', before, lengthAfterAdd:collection.length, mapPrototypeChanged:Object.getPrototypeOf(collection.map) !== Object.prototype, phantomLookup:collection.key('phantom')});
      for (const query of ['__proto__=x', 'constructor=x', 'toString=x']) {
        let error = null;
        try {Ext.urlDecode(query);} catch(e) {error = String(e);}
        observations.push({name:'urlDecode prototype key', query, error});
      }
      for (const value of [Ext.decode('{"hasOwnProperty":"data","safe":1}'), Object.assign(Object.create(null),{safe:1})]) {
        let output, error = null;
        try {output = Ext.encode(value);} catch(e) {error = String(e);}
        observations.push({name:'JSON encoder shadowed/missing hasOwnProperty',output,error});
      }
      const records = new Ext.data.JsonStore({id:'id',root:'rows',fields:['name'],data:{rows:[{id:'__proto__',name:'Ada'}]}});
      observations.push({name:'Store response id __proto__',count:records.getCount(), phantomDataLookup:records.getById('data')?.name});
      delete window.__htmlAudit;
      return observations;
    });
    report.push({distribution, observations});
    await page.close();
  }
} finally {
  await browser.close();
  await new Promise(resolve => server.close(resolve));
}
const output = JSON.stringify(report, null, 2) + '\n';
if (process.argv[2]) await writeFile(process.argv[2], output);
else process.stdout.write(output);
