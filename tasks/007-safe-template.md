# 007: Nonexecuting text templates

Status: complete

Authorized by the user: strengthen templating features during the security audit.

## Decisions

- Add opt-in Ext.SafeTemplate in the shared source patch layer, preserving
  existing Template/XTemplate APIs and expression behavior.
- Support text placeholders and dotted own-data paths, including numeric array
  segments, with DOM text binding and serialization instead of eval/Function.
- Reject getters, object coercion, prototype paths, formatters, and expressions.
- Validate a restricted static HTML subset; reject attribute bindings, URLs,
  style, executable elements, and event attributes.
- Insert fragments with DOM operations and reject active insertion containers.
- Document the supported subset and compatibility boundaries; verify both
  distribution formats, packaged output, and string-compilation restrictions.

## Validation

- Shared patches add the class to classic, debug, core, and module distributions;
  upstream files remain unchanged.
- Packaged Chromium checks passed for native/late imports, classic minified and
  debug, Vite development, and Vite production under /demo/.
- Checks cover escaped markup, own-data paths, arrays, missing/null fields,
  literal braces, serialization, fragment freshness, insertion order/wrappers,
  invalid-template and invalid-render preservation, malicious expressions,
  active elements/attributes, prototype paths, getters, coercion, and functions.
- Constructor/compile/render/insertion work with global eval and Function blocked
  after framework initialization. Full-library CSP compatibility is not claimed.
- `go test ./...`, `go vet ./...`, browser-script syntax checks, and
  `git diff --check` passed. The rebuilt native CLI generated 191 API classes,
  including Ext.SafeTemplate, and the website includes the audit and usage guide.
- Firefox remains for its Ubuntu CI job; the local cached Firefox executable
  has the launch failure documented in the preceding task.

## Integration limits

This is an opt-in text/layout template subset. It has no loops, expression
conditions, formatter dispatch, attribute bindings, or URL/style support.
Legacy Template/XTemplate and existing widget defaults remain available with
their documented trust boundaries. SafeTemplate is not a general sanitizer or
a drop-in replacement for APIs that require an XTemplate instance.
