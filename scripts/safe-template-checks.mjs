export function runSafeTemplateChecks(Ext) {
  const check = (ok, message) => {if (!ok) throw new Error('SafeTemplate regression: ' + message);};
  function rejects(run, message) {
    let rejected = false;
    try {run();} catch {rejected = true;}
    check(rejected, message);
    check(window.__safeTemplateAudit === 0, message + ' executed code');
  }
  window.__safeTemplateAudit = 0;
  const attack = '<img src="data:image/png;base64,invalid" onerror="window.__safeTemplateAudit=1">';
  const host = document.createElement('div');
  document.body.appendChild(host);
  try {
    const template = new Ext.SafeTemplate('<p class="safe-template">Hello {person.name}, {items.0}, {count}, {flag}, {missing.child}! {{literal}}</p>').compile();
    const data = {person:{name:attack},items:['first'],count:0,flag:false};
    template.append(host, data);
    check(host.textContent === 'Hello ' + attack + ', first, 0, false, ! {literal}', 'text binding and escaped braces');
    check(!host.querySelector('img') && host.querySelector('p').className === 'safe-template', 'data cannot create markup');
    const serialized = template.apply(data);
    check(serialized.includes('&lt;img') && serialized.includes('&gt;'), 'serialized data escaped');
    check(template.applyTemplate(data) === serialized, 'applyTemplate alias');
    check(new Ext.SafeTemplate('{name}').apply({name:'{[window.__safeTemplateAudit=1]}'}) === '{[window.__safeTemplateAudit=1]}', 'no second template pass');
    check(new Ext.SafeTemplate('{name}').apply(Object.create({name:'inherited'})) === '', 'own data only');
    check(new Ext.SafeTemplate('{person.name}').apply({person:null}) === '', 'null intermediate');
    check(new Ext.SafeTemplate('{0}').apply(['array']) === 'array', 'numeric root');
    for (const invalid of [
      '{[window.__safeTemplateAudit=1]}', '{name:htmlEncode}', '{name:call("attack")}', '{constructor}', '{person.__proto__}', '{person.prototype}',
      '<tpl if="name">{name}</tpl>', '<script>window.__safeTemplateAudit=1</script>', '<style>{name}</style>', '<svg onload="window.__safeTemplateAudit=1"></svg>',
      '<iframe srcdoc="attack"></iframe>', '<img src="data:image/png;base64,invalid" onerror="window.__safeTemplateAudit=1">',
      '<div onclick="window.__safeTemplateAudit=1">safe</div>', '<div title="{name}">safe</div>', '<div class="{name}">safe</div>',
      '<div style="color:red">safe</div>', '<a href="javascript:attack()">safe</a>', '<!-- {name} -->', '{', '{}', '}'
    ]) rejects(() => new Ext.SafeTemplate(invalid), 'unsupported template ' + invalid);
    const getters = {};
    Object.defineProperty(getters, 'name', {get(){window.__safeTemplateAudit=1;return 'attack';}});
    rejects(() => new Ext.SafeTemplate('{name}').apply(getters), 'data getter');
    rejects(() => new Ext.SafeTemplate('{name}').apply({name:{toString(){window.__safeTemplateAudit=1;return 'attack';}}}), 'object coercion');
    rejects(() => new Ext.SafeTemplate('{name}').apply({name:()=>{window.__safeTemplateAudit=1;}}), 'function value');
    const previous = template.apply(data);
    rejects(() => template.set('{name:call()}'), 'invalid set');
    check(template.apply(data) === previous, 'invalid set preserves template');
    rejects(() => new Ext.SafeTemplate('{name}').overwrite(host, getters), 'invalid render');
    check(host.textContent.includes(attack), 'invalid overwrite preserves DOM');

    const layout = new Ext.SafeTemplate('<span>{value}</span>');
    host.textContent = '';
    const middle = layout.append(host, {value:'middle'}, true);
    check(middle.dom.textContent === 'middle', 'element wrapper return');
    layout.insertBefore(middle, {value:'before'});
    layout.insertAfter(middle, {value:'after'});
    layout.insertFirst(host, {value:'first'});
    check(host.textContent === 'firstbeforemiddleafter', 'DOM insertion order');
    layout.overwrite(host, {value:attack});
    check(host.textContent === attack && !host.querySelector('img'), 'overwrite data as text');
    const fragment = layout.render({value:'fragment'});
    check(fragment.nodeType === 11 && fragment.textContent === 'fragment' && layout.render({value:'again'}) !== fragment, 'fresh fragments');
    for (const tag of ['script','style','iframe','svg']) {
      rejects(() => layout.append(document.createElement(tag), {value:'attack'}), 'active insertion container');
    }
    const originalEval = window.eval, OriginalFunction = window.Function;
    window.eval = window.Function = function(){throw new Error('String compilation blocked');};
    try {
      new Ext.SafeTemplate('<b>{name}</b>').compile().overwrite(host, {name:attack});
      check(host.textContent === attack && !host.querySelector('img'), 'works without string compilation');
    } finally {
      window.eval = originalEval;
      window.Function = OriginalFunction;
    }

    check(Ext.util.Format.htmlEncode("'\"<>&") === '&#39;&quot;&lt;&gt;&amp;', 'both attribute quotes encoded');
    check(Ext.util.Format.htmlDecode('&#39;&#x27;&apos;') === "'''", 'apostrophe decoding');
    const legacy = new Ext.Template("<span title='{name:htmlEncode}'>safe</span>");
    legacy.overwrite(host, {name:"' onmouseover='window.__safeTemplateAudit=1"});
    check(!host.firstChild.hasAttribute('onmouseover'), 'legacy single-quoted attribute encoding');
    check(Ext.encode(Ext.decode('{"hasOwnProperty":"data","safe":1}')).includes('"hasOwnProperty":"data"'), 'JSON encoder shadowed ownership method');
    check(Ext.encode(Object.assign(Object.create(null), {safe:1})) === '{"safe":1}', 'null-prototype JSON encoding');
    check(window.__safeTemplateAudit === 0, 'untrusted data never executed');
  } finally {
    host.remove();
    delete window.__safeTemplateAudit;
  }
}
