# HorizonJS

A rebuild of Ext JS 2.0.2 — the last open-source release — updated to build cleanly and run on modern browsers with minor fixes.

## What This Is

Ext JS 2.0.2 (2008) was the final version released under the LGPL 3.0 license before the project went commercial. HorizonJS preserves that release in a form that builds and runs today. The original source, API documentation, and examples are included, with deliberate fixes for contemporary browsers and unsafe runtime input handling.

## Quick Start

Requires Go 1.22 or newer. Library generation, API docs, packaging, and website
assembly run natively in Go. Compiled binaries need no additional build runtime.

All platforms:
```
go run ./cmd/horizon build --tag main
```

To use the original bundled documentation snapshot:
```
go run ./cmd/horizon build --tag main --docs bundled
```

This produces `build/` with:
- `build/` — the concatenated and minified JavaScript library (`ext-all.js`, `ext-core.js`, CSS, images)
- `docs/` — API documentation generated natively from the JavaScript source
- `examples/` — all original Ext JS 2.0.2 examples
- `npm/` — a packable browser module, classic bundles, CSS, images, and licenses
- `website/` — a self-contained site linking everything together (when built with `main`)

To serve locally:
```
./scripts/serve.sh    # starts a server at http://localhost:6969
```

## Browser module package

Each build creates `build/npm`. To make a local installation tarball:

```sh
npm pack ./build/npm --pack-destination ./build
```

Install that tarball in your application, then use:

```js
import Ext from '@horizonjs/core';
import '@horizonjs/core/styles.css';

Ext.onReady(function () {
    new Ext.Panel({ renderTo: 'app', title: 'Hello', width: 320 });
});
```

`Ext` is also a named export and remains available as `window.Ext` for plugins.
The standalone adapter initializes before widgets. Bundlers resolve CSS images
and the module's blank-image URL automatically. Native browsers can import
`index.mjs` from a served package directory and load CSS with a stylesheet link.
See [the package guide](overlay/npm/README.md) for native imports, themes, legacy
files, and browser-only usage. The package name is provisional; publishing is
not configured by this change.

Shared patches in `overlay/source/patches.json` harden classic and module builds
before minification. Module-specific patches in `overlay/module/patches.json`
then handle strict-mode compatibility. The archived `extjs-2.0.2/` files remain
unchanged, and a patch that no longer matches fails the staged build.
See [runtime input rules](RUNTIME_SECURITY.md) for strict JSON, property mappings,
provider registration, selectors, and trusted templates. Semantic-version tags such as `v1.0.0`
set the package version; other builds use `0.1.0-dev.0`.

For ordinary data display, use `new Ext.SafeTemplate('<p>{person.name}</p>')`.
It binds text without executing expressions or interpreting inserted data as
markup. Legacy Template/XTemplate remain available for trusted definitions.
See [the HTML/prototype audit](HTML_PROTOTYPE_AUDIT.md) for remaining widget
rendering and object-copying findings.

To run the package checks (Node 22.12+ and Go required):

```sh
npm ci
npx playwright install chromium
npm run test:browser
```

The checks install the actual npm tarball, validate every CSS image reference,
and exercise native imports, imports after page load, classic scripts, Vite
development, and Vite production under a subpath. Widgets, forms, date pickers,
templates, stores, events, and animations are covered. For Firefox, install its
Playwright browser and run `npm run test:browser:firefox`.

## Build Process

Running `go run ./cmd/horizon build --tag <tag>` executes the following pipeline.
The existing `./build.sh <tag>` and `./build.ps1 <tag>` commands forward to this
same tool. See [BUILD_TOOL.md](BUILD_TOOL.md) for flags, compatibility rules,
failure handling, and validation.

### 1. JSBuild (JavaScript Concatenation)

The Go tool reads two `.jsb` project files from the original Ext JS source:

- `source/ext.jsb` — concatenates source files in dependency order into `ext-all.js` and `ext-core.js`
- `resources/resources.jsb` — copies CSS and images into the output

The original JSMin behavior is preserved. Output is generated in a staging
directory and published to `build/build/` only after the entire build succeeds.
No .NET runtime or JSBuild container is needed.

### 2. Extdoc (API Documentation)

The native Go generator in `internal/extdoc/` reads documentation comments from
the freshly built source and renders class pages, inheritance, members, source
links, and navigation data. The existing docs UI assets are preserved. All 190
class pages are checked against a Java reference baseline, allowing HTML
serialization whitespace differences. Source pages escape code so literal HTML
remains visible.

`--docs go` is the default. Documentation UI assets live in `assets/docs/`.
`--docs bundled` copies the original checked-in docs snapshot.

### 3. Examples

The original `extjs-2.0.2/examples/` directory is copied verbatim into `build/examples/`.

### 4. Cleanup

Intermediate build artifacts are removed: raw source files, unused CSS themes, license files that don't apply to the built artifact, and the original `.jsb` project files.

### 5. Overlay (Patches and Branding)

The `overlay/` directory is layered on top of the build output:

```
cp -Rf ./overlay/build/.  ./build/build/
cp -Rf ./overlay/docs/.   ./build/docs/
cp -Rf ./overlay/examples/. ./build/examples/
cp -Rf ./overlay/website/. ./build/website/
```

**Purpose:** The overlay is where all modifications to HorizonJS live. Instead of altering files inside `extjs-2.0.2/` (which would muddy the fork's provenance and make merging future fixes harder), changes are placed in `overlay/` and copied over the build output. This keeps the original Ext JS 2.0.2 source pristine while allowing:

- Bug/security fixes to JavaScript and CSS for modern browser compatibility
- New or patched documentation pages
- Website structure and release artifacts

To fix a bug, add the corrected file to `overlay/build/` mirroring its path in the build output. It will overwrite the original during the overlay step.

### 6. Archive

The validated library, docs, and examples are zipped with Go's built-in ZIP
writer into versioned archives under `build/zips/`. Command, filesystem, and
archive errors fail the build with a nonzero exit code. Failed builds preserve
the previous output directory.

### 7. Website (main tag only)

When the tag is `main`, a self-contained website is assembled at `build/website/` combining the library, docs, examples, release zips, and the overlay website template.

## What's Changed

HorizonJS is a faithful rebuild of Ext JS 2.0.2. The only differences from the original:

- The proprietary JSBuild GUI tool has been replaced with a Go build CLI (`cmd/horizon/`); the previous .NET CLI and container have been removed
- Library generation, documentation, packaging, and website assembly run in Go; the original Java/Ant generator and container have been removed
- Minor patches for modern browser compatibility (CSS, DOM API changes)
- `Ext` namespace and branding updated throughout

The Ext 2 constructor API remains available. Security fixes deliberately require
strict JSON responses, data-only reader mappings, and registered tree UI provider
aliases. Applications relying on executable response data must migrate; see
[RUNTIME_SECURITY.md](RUNTIME_SECURITY.md).

## License

The original Ext JS 2.0.2 JavaScript source is LGPL 3.0. CSS, images, and other assets are licensed under their original terms (see `extjs-2.0.2/LICENSE.txt`).
