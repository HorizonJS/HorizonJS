# HTML insertion and prototype audit

Audited on 2026-10-13 against the generated module, classic minified, and classic
debug distributions with the earlier JSON/selector fixes. The manual browser
harness uses harmless markers, data URLs, and blocks non-local requests.
Exploitation requires control of data passed through the relevant API.

## Confirmed open findings

| Priority | Input and sink | Verified effect | Recommended change |
| --- | --- | --- | --- |
| High | JSON store field to default grid renderer; `widgets/grid/ColumnModel.js:550`, `GridView.js:512` | An image error handler became an element and executed. | Escape default text; retain rich markup through explicit application renderers. |
| High | JSON tree text to `widgets/tree/TreeNodeUI.js:438` | Initial text became executable HTML. Updates also use innerHTML at line 52. | Literal text by default, with explicit trusted rich markup. |
| High | Tree attributes to `TreeNodeUI.js:438` | A quote in `cls` injected an event attribute, executed when dispatched. IDs/icons/targets/hrefs use the same concatenation. | DOM attribute assignment or attribute encoding, with name and URL validation. |
| High | Tree `href` at `TreeNodeUI.js:437` | A javascript URL executed on anchor activation. The test exercised the renderer directly, without TreePanel click interception. | Define allowed schemes and navigation behavior; encoding does not validate URLs. |
| High | Form response errors through BasicForm to `widgets/form/Field.js:403` | A valid JSON error message executed as HTML in a custom msgTarget. | Text messages by default; explicit trusted markup for rich messages. |
| High | DOM spec title to `core/DomHelper.js:57` | A quote in an ordinary attribute value injected an event handler. | Escape values, validate names, and document explicit HTML/handler trust boundaries. |
| High | JSON object to `Ext.apply`; `core/Ext.js:29` | An own __proto__ key changed the receiver's prototype and introduced an inherited option. | Prevent prototype mutation; define own-data copying semantics. |
| High | Store ID to MixedCollection map; `util/MixedCollection.js:26`, `:85` | ID __proto__ dropped the visible row count to zero and exposed record data through a phantom key. | Null-prototype maps and own-key membership throughout map lifecycle and lookup. |
| Medium | Inherited options to apply/applyIf; `core/Ext.js:35`, `:127` | Both copied inherited values as own destination options. | Inventory inherited configuration dependencies before changing this broad API. |
| Medium | URL/form names to `Ext.urlDecode`; `core/Ext.js:351` | __proto__, constructor, and toString names throw because inherited values are treated as arrays. | Null-prototype results and own-key checks, preserving repeated-name behavior. |

Paths above are relative to `extjs-2.0.2/source/`. The copy test mutated its
receiver only: Object.prototype itself remained unchanged. A global prototype
pollution chain was not demonstrated.

## Related paths inspected

Template.overwrite and Element.update are deliberate HTML APIs. Passing
untrusted strings remains unsafe with loadScripts false, because event
attributes do not need script tags. Grid header tooltips concatenate attributes
at GridView.js:745; tree qtips reach QuickTip.body.update at tips/QuickTip.js:154.
Form setValues/markInvalid enumerate objects without own checks. Component
configuration/state and tree/node registries also deserve copy/map review.
These related paths were inspected but not all independently exercised.

Reader path guards protect lookup; they do not secure later copying, dictionary
insertion, or rendering of successfully parsed JSON.

## Corrections made during this pass

- Added opt-in Ext.SafeTemplate through shared patches, with own-data text
  bindings and a restricted static HTML subset. It rejects executable syntax,
  attribute bindings, active elements/attributes, prototype paths, getters, and
  object coercion. DOM insertion uses fragments.
- htmlEncode now escapes apostrophes as well as ampersands, angle brackets, and
  double quotes, with corresponding apostrophe decoding. The single-quoted
  attribute test now creates no event handler in all three distributions.
- JSON encoding uses the intrinsic ownership checker. A hasOwnProperty data
  field and null-prototype records now encode correctly. This does not make
  subsequent copying of arbitrary JSON keys safe.

## Next compatibility decisions

Start with map correctness and ordinary attribute escaping. Follow with text
defaults for grid values, tree labels, and validation messages, preserving
intentional rich markup through explicit application configuration. Replacing
every innerHTML call or all inherited-copy behavior would alter useful legacy
features. Keep text encoding, attribute encoding, URL validation, and key
handling separate.

SafeTemplate is an additive migration path. Legacy widget rendering, copying,
and collection maps remain open findings. Its static structure must be
application-authored; arbitrary JavaScript objects/proxies are outside its
ordinary-data contract.

## Reproduction and validation

```sh
npm run build
node scripts/audit-html-prototypes.mjs
# Optional report file
node scripts/audit-html-prototypes.mjs audit-results.json
```

The harness reports observations instead of requiring vulnerabilities to remain
present. It is separate from CI, where packaged SafeTemplate/helper checks run
with native/late module imports, classic minified/debug, and Vite development
and production. Legacy executable templates, full-library CSP compatibility,
and the separate historic docs runtime remain documented boundaries.

References: [HTML insertion rules](https://html.spec.whatwg.org/multipage/dynamic-markup-insertion.html),
[receiver mutation and prototype pollution](https://developer.mozilla.org/en-US/docs/Web/Security/Attacks/Prototype_pollution).
