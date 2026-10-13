# Browser module patches

The Go build reads the original source targets from the manifest and emits a
separate strict-mode ES module in `build/npm/index.mjs`. It applies the ordered,
counted replacements in patches.json; missing matches or unused patches fail
the build. Classic bundles and the upstream checkout retain their bytes.

The patches address:

- `Ext` initialization and the read-only `window.undefined` property.
- Namespace creation without implicit globals or declaration through eval.
- Selector compiler locals that strict eval cannot create in its caller.
- Named animation callbacks replacing `arguments.callee`, retaining `.anim`
  references for `stopFx` and queued/concurrent effects.
- Explicit native event arguments instead of inspecting function callers.
- DOM readiness through native events, including imports after page load.
- String-form event registration writing to the Observable's event map.
- Local variables in the date picker and drag/drop handlers.

The module keeps `window.Ext` for plugin and generated-code compatibility.
It rejects imports without a browser DOM and duplicate distribution loads.
It retains upstream prototype extensions and dynamic code generation.
