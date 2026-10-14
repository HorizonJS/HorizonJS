# 006: HTML insertion and prototype audit

Status: complete

Authorized by the user: audit HTML insertion and prototype handling.

- Trace application data through HTML, attributes, URLs, copying, and maps.
- Reproduce exposures using harmless local markers in generated distributions.
- Distinguish receiver prototype mutation from global prototype pollution.
- Record concrete findings, priorities, and compatibility implications.
- Retain a manual audit harness; do not put assertions requiring vulnerabilities
  to remain present in normal CI.

## Result and validation

- Recorded ten confirmed open categories and related inspected sinks in
  HTML_PROTOTYPE_AUDIT.md, with locations and compatibility decisions.
- Reproduced observations in Chromium for module, classic minified, and classic
  debug output. The final harness blocks non-local requests and uses harmless
  data-URL/event markers.
- Demonstrated receiver prototype mutation, not global Object.prototype
  pollution; demonstrated store row loss/phantom lookup from a __proto__ ID.
- Corrected apostrophe encoding and JSON ownership-method shadowing through
  shared patches. Reruns confirmed no injected event attribute and successful
  encoding of null-prototype and hasOwnProperty-bearing objects.
- The remaining rendering/copy/map findings are open, not marked as fixed.
  The additive SafeTemplate work is tracked in task 007.
