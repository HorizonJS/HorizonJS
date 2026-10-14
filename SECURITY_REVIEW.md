# Runtime code execution review

Investigated and hardened on 2026-10-13. This records the original findings;
shared source patches now protect the generated classic and module bundles.
See [RUNTIME_SECURITY.md](RUNTIME_SECURITY.md) for current input rules and the
remaining trusted-template, HTML, and CSP boundaries.

## Verified findings

The pre-hardening classic and browser module distributions both executed the harmless
marker `window.__probe = 1` through the following inputs. These checks ran in
local Chromium against the generated distributions, without external requests.

| Input boundary | Implementation | Finding |
| --- | --- | --- |
| JSON text | `source/util/JSON.js:131` | `Ext.decode` evaluates JavaScript expressions instead of enforcing JSON syntax. |
| Store response text | `source/data/JsonReader.js:100` | `JsonReader.read` has its own eval, bypassing `Ext.decode`. |
| Tree response text | `source/widgets/tree/TreeLoader.js:226` | `TreeLoader.processResponse` also has its own eval. |
| Reader metadata in valid JSON | `source/data/JsonReader.js:133` | Server-supplied field mappings, root, ID, and total/success property paths can enter a `new Function` accessor. |
| Tree attributes in valid JSON | `source/widgets/tree/TreeLoader.js:216` | An unregistered string `uiProvider` is evaluated as JavaScript. |
| Selector text | `source/core/DomQuery.js:437` | Captured selector tokens enter generated JavaScript without string-literal escaping. A crafted pseudo argument executes code. |
| Basic template definition | `source/core/Template.js:118` | Compiled formatter arguments can contain executable expressions. |
| Advanced template definition | `source/util/XTemplate.js:194` | Conditions, `exec`, `for`, and inline expressions deliberately execute template-authored code. |

Source paths above are relative to the unchanged archive `extjs-2.0.2/`.
The original module-only patches fixed strict-mode execution. Shared source
patches now enforce strict response parsing, data-only reader accessors,
registered providers, and literal selector tokens before either format is built.

The response-related findings become code execution when an application passes
attacker-controlled response content through these APIs. Selector injection
requires control of selector text or a value concatenated into a selector.
Executable templates require control of the template definition. Ordinary
string values supplied to a trusted template are not recompiled as JavaScript.
Template output is nevertheless unescaped HTML unless an appropriate formatter
is used; DOM insertion is a separate injection boundary.

## Minimal local reproductions

Reset `window.__probe = 0` before each case. These payloads only set a local
marker; they do not transmit information or modify files.

```js
Ext.decode('(window.__probe=1,{"ok":true})');

new Ext.data.JsonReader({root: 'rows'}, []).read({
    responseText: '(window.__probe=1,{"rows":[]})'
});

// Valid JSON still reaches executable field-mapping configuration.
new Ext.data.JsonReader({root: 'rows'}, []).read({
    responseText: JSON.stringify({
        metaData: {
            root: 'rows',
            fields: [{name: 'name', mapping: 'name[window.__probe=1]'}]
        },
        rows: [{name: ['zero', 'one']}]
    })
});

new Ext.tree.TreeLoader().createNode({
    leaf: true,
    uiProvider: '(window.__probe=1,Ext.tree.TreeNodeUI)'
});

Ext.query('*:contains("+(window.__probe=1)+")');

new Ext.Template('{name:ellipsis((window.__probe=1))}')
    .compile().apply({name: 'safe'});

new Ext.XTemplate('{[window.__probe=1]}').apply({});
```

The attribute form `*[title='x"+(window.__probe=1)+"']` also executed the marker
when the original DomQuery was tested in an isolated Node VM with an empty DOM
stub. The browser check independently confirmed the pseudo form above in both
distributions.

## Implemented hardening decisions

1. Treat JSON and response metadata as data throughout the pipeline. Use
   `JSON.parse` for the public decoder and route reader/tree response parsing
   through it. Do not fall back to eval on a parse failure. Keep the current
   encoder's date and array behavior as a separate compatibility decision.
2. Replace generated reader accessors with a parser for property paths. Preserve
   normal dotted names, numeric array indexes, and quoted bracket keys. Reject
   calls, assignments, operators, and other expressions. Define missing-path
   behavior and inherited-property access explicitly; test before narrowing
   historical behavior. Register tree UI providers as application-controlled
   aliases and reject unknown names, rather than evaluating a response string.
3. Fix selector token injection while preserving Ext's operators, pseudos, and
   XPath-like syntax. Escape all captured values for the generated string
   literals, or interpret parsed tokens. Custom matcher code remains trusted
   application code. Native `querySelectorAll` alone is not a compatible
   replacement for the extended selector grammar.
4. Keep executable Template/XTemplate definitions restricted to trusted
   application code for the initial hardening patch. Document the boundary and
   retain their existing expression features. A nonexecuting template subset
   needs a distinct API or mode with an explicit unsupported-feature policy.
   Data inserted as HTML still requires context-appropriate escaping.

Security fixes apply to both classic and module distributions through shared,
count-checked source patches before compilation. Original `extjs-2.0.2/` files
and upstream builder baseline hashes remain unchanged. Production behavior is
verified separately with packaged browser exploit regressions and compatibility
checks. Executable templates remain a deliberate trusted-code feature.

## Compatibility and validation

Strict JSON parsing rejects the current `examples/tree/column-data.json` and
`examples/grid/summary.json`, which use unquoted keys and single-quoted strings.
`examples/dependency/dep-tree.json` is valid JSON. Affected distribution feeds
are converted through overlays, without changing the archived upstream source.
The checked-in tree PHP endpoint already uses `json_encode`; custom endpoints
and historical documentation examples need the migration guidance in
`RUNTIME_SECURITY.md`.

Regression tests should exercise both output formats and all network readers,
prove malformed/expression input has no side effects, cover valid JSON scalars
and nested data, validate metadata paths and registered providers, and verify
selector literals containing quotes and backslashes. Preserve representative
trusted template expression behavior and document the HTML escaping boundary.

Replacing eval with `new Function` does not remove code execution or satisfy
a CSP that blocks string compilation. Strict JSON improves these data paths,
but templates, selectors, date formatting/parsing, and Format helpers still use
runtime compilation. Full operation without `unsafe-eval` requires a broader
compatibility project.

References: [ECMAScript JSON.parse](https://tc39.es/ecma262/multipage/structured-data.html#sec-json.parse),
[CSP string compilation rules](https://www.w3.org/TR/CSP3/#can-compile-strings).
