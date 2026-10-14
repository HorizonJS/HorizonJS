# 005: Runtime data and selector hardening

Status: complete

Authorized by the user: fix the findings in SECURITY_REVIEW.md using the patch
system where possible.

## Decisions and scope

- Extend counted source patches to run before classic minification, source
  staging, and module assembly, keeping archived upstream files unchanged.
- Parse JSON strictly and route JsonReader and TreeLoader through Ext.decode.
- Interpret reader property paths with own-property lookup; reject executable
  expressions and prototype traversal. Missing paths return undefined.
- Resolve string UI providers only through registered application aliases.
- Escape selector tokens before inserting them into generated string literals.
- Preserve executable Template/XTemplate definitions as trusted application
  code; document HTML escaping and the remaining CSP constraint.
- Convert affected static example feeds through overlays; retain original
  builder baselines independently of intentionally changed build output.
- Verify failure handling and exploit prevention in classic and module outputs,
  packaged browser tests, and CI.

## Validation

- `go test ./...` and `go vet ./...` passed. Tests preserve unpatched upstream
  builder hashes separately, verify shared patches reach classic/core/module
  outputs, and cover source drift, unused patches, replacement-file errors,
  unsafe paths, and preservation of previous output on patch failure.
- `npm run test:browser` passed packaged security and compatibility checks in
  Chromium for native imports, late imports, classic minified, classic debug,
  Vite development, and production under `/demo/`.
- Browser regressions cover strict JSON and form/reader/tree parsing, malicious
  valid-JSON metadata, provider aliases, prototype access, selector injection,
  quote/backslash/control/Unicode separator literals, trusted template behavior,
  field defaults, metadata failure state, and tree failure/success events.
- Browser checks prove JSON decoding and reader accessors still work when the
  global eval/Function entry points are blocked after library initialization.
  This does not assert full-library CSP compatibility.
- Static example overlays preserve the original data while providing strict
  JSON. The source documentation for mappings and tree responses is patched.
- Full website and versioned release builds passed with native documentation,
  all three ZIPs, and the runtime guide included in the website.
- Updated workflow passed native actionlint; both browser scripts passed Node
  syntax checks. Firefox remains blocked by the cached Windows executable's
  launch failure; its existing Ubuntu CI job runs the new security checks too.
- Original `extjs-2.0.2/` files remain unchanged; `git diff --check` passed.

## Remaining boundaries

Template definitions and custom matcher snippets remain trusted application
code. Unformatted HTML, explicit script loading, and the historical docs UI's
separate runtime bundle are not sanitized or rewritten by this patch. Full
operation without CSP string compilation requires a separate runtime project.
