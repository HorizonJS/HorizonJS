// Runs inside a browser against the installed package in every distribution mode.
export function runSecurityChecks(Ext) {
  function check(ok, message) {
    if (!ok) throw new Error('Security regression: ' + message);
  }
  function rejects(run, message) {
    let rejected = false;
    try { run(); } catch { rejected = true; }
    check(rejected, message);
    check(window.__horizonSecurityProbe === 0, message + ' executed code');
  }
  window.__horizonSecurityProbe = 0;
  const expression = '(window.__horizonSecurityProbe=1,{"rows":[]})';
  for (const input of ['null', 'true', 'false', '0', '-1.25e3', '"hello"', '{"items":[1,null,"é"],"line":"a\\nb"}']) {
    check(JSON.stringify(Ext.decode(input)) === JSON.stringify(JSON.parse(input)), 'valid JSON ' + input);
  }
  for (const input of [expression, '', '{a:1}', "{'a':1}", '[1,]', '/*comment*/{}', 'NaN', 'Infinity', 'undefined', '1+1', 'new Date()']) {
    rejects(() => Ext.decode(input), 'non-JSON decoder input');
  }
  const parsed = Ext.decode('{"__proto__":{"polluted":true}}');
  check(Object.prototype.hasOwnProperty.call(parsed, '__proto__') && Object.getPrototypeOf(parsed) === Object.prototype && !({}).polluted, 'JSON prototype semantics');

  const reader = new Ext.data.JsonReader({root:'payload.rows', id:'ids[0]', totalProperty:'counts.total', successProperty:'flags.ok'}, [
    {name:'name', mapping:'details["display.name"]'},
    {name:'missing', mapping:'missing.child', defaultValue:'fallback'}
  ]);
  const data = reader.read({responseText:JSON.stringify({payload:{rows:[{ids:['id-1'], details:{'display.name':'Ada'}}]}, counts:{total:12}, flags:{ok:true}})});
  check(data.records[0].id === 'id-1' && data.records[0].get('name') === 'Ada' && data.records[0].get('missing') === 'fallback' && data.totalRecords === 12, 'nested reader paths');
  const accessor = expr => reader.getJsonAccessor(expr);
  const values = {rows:[{name:'Ada', 'a-b':7}], 'first name':'Grace', "quo'te":8, 'slash\\key':9, 'line\nkey':10};
  for (const [path, expected] of [['rows[0].name','Ada'], ["rows[0]['a-b']",7], ['["first name"]','Grace'], ["['quo\\'te']",8], ['["slash\\\\key"]',9], ['["line\\nkey"]',10]]) {
    check(accessor(path)(values) === expected, 'literal accessor ' + path);
  }
  check(accessor(0)(['zero']) === 'zero', 'numeric array mapping');
  check(accessor('a.b')({a:null}) === undefined && accessor('value')(Object.create({value:'inherited'})) === undefined, 'missing and inherited paths');
  for (const path of ['name[window.__horizonSecurityProbe=1]', 'a..b', 'a()', 'a;window.__horizonSecurityProbe=1', '__proto__', 'a.constructor', 'a["prototype"]', '["\\u005f_proto__"]', "['\\x61']"]) {
    rejects(() => accessor(path), 'unsafe accessor ' + path);
  }
  rejects(() => reader.read({responseText:expression}), 'reader response expression');
  for (const property of ['root','id','totalProperty','successProperty']) {
    const originalMeta = reader.meta;
    let changed = false;
    reader.onMetaChange = () => { changed = true; };
    const metaData = {root:'rows', fields:['name']};
    metaData[property] = 'rows[window.__horizonSecurityProbe=1]';
    rejects(() => reader.read({responseText:JSON.stringify({metaData,rows:[]})}), 'malicious metadata ' + property);
    check(reader.meta === originalMeta && !changed, 'metadata failure changed reader state');
  }
  for (const field of [{name:'name',mapping:'name[window.__horizonSecurityProbe=1]'}, {name:'constructor'}, JSON.parse('{"name":"name","__proto__":{}}')]) {
    rejects(() => reader.read({responseText:JSON.stringify({metaData:{root:'rows',fields:[field]},rows:[{name:[]} ]})}), 'malicious field metadata');
  }
  const changing = new Ext.data.JsonReader({root:'rows'}, []);
  const updated = changing.read({responseText:JSON.stringify({metaData:{root:'rows',fields:[{name:'label',mapping:'person.name'}]},rows:[{person:{name:'Grace'}}]})});
  check(updated.records[0].get('label') === 'Grace', 'valid server metadata');
  for (const Action of [Ext.form.Action.Submit, Ext.form.Action.Load]) {
    rejects(() => Action.prototype.handleResponse.call({form:{}}, {responseText:expression}), 'form response expression');
    check(Action.prototype.handleResponse.call({form:{}}, {responseText:'{"success":true}'}).success === true, 'valid form response');
  }

  const provider = Ext.extend(Ext.tree.TreeNodeUI, {});
  const loader = new Ext.tree.TreeLoader({uiProviders:{registered:provider}});
  const child = loader.createNode({leaf:true, text:'safe', uiProvider:'registered'});
  check(child.ui instanceof provider, 'registered UI provider');
  rejects(() => loader.createNode({leaf:true,uiProvider:'(window.__horizonSecurityProbe=1,Ext.tree.TreeNodeUI)'}), 'provider expression');
  rejects(() => loader.createNode({leaf:true,uiProvider:'constructor'}), 'inherited provider name');
  const inheritedProvider = new Ext.tree.TreeLoader({uiProviders:Object.create({registered:provider})});
  rejects(() => inheritedProvider.createNode({leaf:true,uiProvider:'registered'}), 'inherited provider registry');
  for (const input of ['(window.__horizonSecurityProbe=1,[])', '{}', '[null]', '[{"leaf":true,"uiProvider":"unknown"}]']) {
    let appended = 0, begin = 0, failures = 0, loads = 0, callbacks = 0;
    const invalid = new Ext.tree.TreeLoader();
    invalid.on('loadexception', () => failures++);
    invalid.on('load', () => loads++);
    const node = {beginUpdate(){begin++;}, endUpdate(){}, appendChild(){appended++;}};
    invalid.handleResponse({responseText:input, argument:{node,callback(){callbacks++;}}});
    check(failures === 1 && loads === 0 && callbacks === 1 && appended === 0 && begin === 0, 'invalid tree response handling');
    check(window.__horizonSecurityProbe === 0, 'tree response executed code');
  }
  let appended = 0, updates = 0, loads = 0;
  const node = {beginUpdate(){updates++;},endUpdate(){updates--;},appendChild(){appended++;}};
  loader.on('load', () => loads++);
  loader.handleResponse({responseText:'[{"leaf":true,"text":"safe","uiProvider":"registered"}]',argument:{node}});
  check(appended === 1 && updates === 0 && loads === 1, 'valid tree response');

  const root = document.createElement('section');
  const element = document.createElement('div');
  const literal = '"+(window.__horizonSecurityProbe=1)+"';
  element.textContent = literal;
  root.appendChild(element);
  check(Ext.query('div:contains(' + literal + ')', root)[0] === element, 'pseudo argument remains literal');
  element.setAttribute('title', 'x' + literal);
  check(Ext.query("div[title='x" + literal + "']", root)[0] === element, 'attribute argument remains literal');
  for (const title of ['a"b\\c', 'a\\u0062', 'a\\"b', 'line\nseparator', 'line\rseparator', 'line\u2028separator', 'line\u2029separator']) {
    element.setAttribute('title', title);
    check(Ext.query("div[title='" + title + "']", root)[0] === element, 'selector quote/backslash round trip');
    check(Ext.DomQuery.filter([element], "div[title='" + title + "']")[0] === element, 'simple selector literal');
  }
  check(Ext.query('constructor', root).length === 0 && Ext.query('__proto__', root).length === 0, 'selector cache prototype keys');
  rejects(() => Ext.query('div:constructor', root), 'inherited pseudo name');
  Ext.DomQuery.pseudos.securityTest = function(nodes, value){
    check(value === literal, 'custom pseudo literal');
    return nodes;
  };
  try {
    check(Ext.query('div:securityTest(' + literal + ')', root)[0] === element, 'custom pseudo compatibility');
  } finally {
    delete Ext.DomQuery.pseudos.securityTest;
  }
  check(window.__horizonSecurityProbe === 0, 'selector input executed code');

  // Existing expression features remain explicit trusted-template behavior.
  check(new Ext.Template('{name:ellipsis(2+2)}').compile().apply({name:'longname'}) === 'l...', 'trusted formatter expression');
  const template = new Ext.XTemplate('<tpl if="age &gt; 1">{[values.name.toUpperCase()]}</tpl>');
  check(template.apply({age:2,name:'Ada'}) === 'ADA', 'trusted XTemplate expression');
  const templateValue = '{[window.__horizonSecurityProbe=1]}';
  check(new Ext.Template('{name}').compile().apply({name:templateValue}) === templateValue, 'template value treated as data');
  check(new Ext.XTemplate('{name}').apply({name:templateValue}) === templateValue, 'XTemplate value treated as data');
  check(new Ext.Template('{name:htmlEncode}').compile().apply({name:'<img onerror="attack()">'}).startsWith('&lt;img'), 'text-context HTML encoding');

  const originalEval = window.eval, OriginalFunction = window.Function;
  window.eval = window.Function = function(){throw new Error('String compilation blocked');};
  try {
    check(Ext.decode('{"safe":true}').safe === true && accessor('rows[0].name')(values) === 'Ada', 'JSON/accessors need no string compilation');
  } finally {
    window.eval = originalEval;
    window.Function = OriginalFunction;
  }
  check(window.__horizonSecurityProbe === 0, 'hardening checks executed untrusted code');
  delete window.__horizonSecurityProbe;
}
