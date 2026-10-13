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

The upstream JavaScript, CSS, and graphics have their original license terms.
See the included LICENSE.txt; asset terms differ from the JavaScript terms.
