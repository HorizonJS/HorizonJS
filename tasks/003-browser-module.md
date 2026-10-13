# 003: Browser module and npm assets

Status: complete

Authorized by the user: add a browser module entry point, compatibility fixes,
and assets usable through the package.

## Scope

- Generate a browser ES module with default and named `Ext` exports.
- Load the standalone adapter before widgets and preserve the classic builds.
- Apply reviewed compatibility patches from the overlay without changing the
  original Ext JS source.
- Produce a packable npm directory containing CSS, images, license notices,
  and documented package exports. Do not publish to a registry.
- Verify native browser imports, Vite development and production, and classic
  script loading, including templates, stores, widgets, events, and animations.

## Validation

- `go test ./...` and `go vet ./...` pass, including module patch drift,
  staged failure preservation, release version mapping, package artifacts,
  and unchanged classic bundle baselines.
- `npm ci` and `npm run test:browser` pass with Chromium. Tests install the
  actual tarball, inspect every CSS asset reference, and exercise native module
  imports, imports after page load, classic scripts, Vite dev, and minified
  Vite production under `/demo/`.
- Runtime checks cover default/named exports, module caching, duplicate-load
  rejection, browser-only import errors, namespaces, compiled templates,
  XTemplate, dates, JSON stores, grids, forms, date pickers, native events,
  drag/drop and splitter handlers, queued animation cancellation, all animation
  callbacks patched for strict mode, and the blank-image asset.
- Firefox checks are available and included in Linux CI. Local Firefox could
  not start because this Windows host lacks its required side-by-side runtime;
  Firefox execution has not been verified locally.
- Vite production emits warnings for upstream direct eval and bundle size.
  Runtime code generation, prototype extensions, browser-only loading, and the
  absence of per-widget tree shaking are documented limitations.

## Implementation

`internal/buildtool/browser.go` generates `build/npm/index.mjs` from the staged
original sources in manifest dependency order. `overlay/module` contains the
module header/footer and counted compatibility patches. `overlay/npm` contains
package metadata and usage instructions. Builds copy classic files, resources,
and upstream license notices alongside the module. Version tags populate the
package version; non-release builds use `0.1.0-dev.0`.

No registry publication or credentials were configured. `@horizonjs/core` is a
provisional package name pending ownership/availability confirmation.
