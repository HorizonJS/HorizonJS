# HorizonJS

Ext JS 2.0.2 rebuilt for modern browsers. This package contains the standalone
adapter, the complete widget library, CSS, and images. It requires a browser DOM.

## Bundled applications

```js
import Ext from '@horizonjs/core';
import '@horizonjs/core/styles.css';

Ext.onReady(function () {
  new Ext.Panel({ renderTo: 'app', title: 'Hello', width: 320, html: 'HorizonJS' });
});
```

Named `import { Ext } from '@horizonjs/core'` also works. The module initializes
the adapter and widgets together and exposes the same object as `window.Ext`
for existing plugins. Load one HorizonJS distribution per page. The package
has initialization side effects and must not be marked as side-effect-free.

Vite and similar bundlers resolve the stylesheet's relative image URLs. The
blank-image URL is set automatically using `new URL(..., import.meta.url)`.
Optionally import `@horizonjs/core/themes/gray.css` after the base stylesheet.

## Native browser modules

Copy the package directory into your site's static files, retaining its layout:

```html
<link rel="stylesheet" href="/horizon/resources/css/ext-all.css">
<div id="app"></div>
<script type="module">
  import Ext from '/horizon/index.mjs';
  Ext.onReady(function () {
    new Ext.Panel({ renderTo: 'app', title: 'Hello', width: 320 });
  });
</script>
```

Serve these files over HTTP. A bare package import in the browser requires an
import map. CSS imports in JavaScript require a bundler; native browser examples
use a stylesheet link.

## Existing script applications

Classic files remain available under the `@horizonjs/core/legacy/*` package
export, including `legacy/adapter/ext/ext-base.js` and `legacy/ext-all.js`.
Copy them to static files and load adapter, library, and CSS as before. Classic
applications still configure `Ext.BLANK_IMAGE_URL` themselves.

## Compatibility

The Ext 2 constructor API is preserved. This is a complete browser bundle,
without per-widget tree shaking or server-side rendering support. In an SSR
application, dynamically import it only from browser code. Prototype extensions
from the original library remain. Some template and selector APIs generate code
at runtime and require a CSP that permits that behavior.

JSON responses are parsed strictly with `JSON.parse`, including stores, trees,
and default form action handlers. Reader mappings accept own-property paths
such as `person.name`, `items[0]`, and `details["display.name"]`; expressions and
prototype traversal are rejected. Missing paths return `undefined` for field
defaults. Register string tree UI providers explicitly in `uiProviders`.
These restrictions apply to both module and classic files.

`new Ext.SafeTemplate('<p>{person.name}</p>')` adds text bindings without runtime
code compilation. It supports dotted own-data paths, numeric array segments,
escaped literal braces, and DOM-fragment insertion. Expressions, formatters,
getters, object coercion, active markup, attribute placeholders, and URL/style/
event attributes are rejected. Keep static markup application-authored.
Existing Template/XTemplate behavior remains available for trusted definitions.

Selector arguments are escaped as literal data. Template definitions retain
their executable expression features and must come from trusted application
code. Use `{name:htmlEncode}` for untrusted text in HTML content; unformatted
values remain HTML. This package does not sanitize arbitrary markup or URLs.

The HTML encoder escapes both quote characters; URL validation remains separate.
Legacy widget HTML sinks and prototype-sensitive copy/collection APIs remain
documented security findings while their migration is underway.

The upstream JavaScript, CSS, and graphics have their original license terms.
See the included LICENSE.txt; asset terms differ from the JavaScript terms.
