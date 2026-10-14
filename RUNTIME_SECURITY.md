# Runtime input rules

The generated classic, debug, core, and browser module bundles share the source
hardening in `overlay/source/patches.json`. Original files in `extjs-2.0.2/` are
archival inputs and remain unchanged. Module-specific strict-mode patches are
applied afterward.

## JSON responses

`Ext.decode` and `Ext.util.JSON.decode` use native `JSON.parse`. JsonReader,
TreeLoader, and the default form action response handlers use the same decoder.
Responses must be valid JSON: double-quoted strings and property names, without
comments, trailing commas, constructors, or expressions. Parse failures do not
fall back to JavaScript evaluation. The existing JSON encoder is unchanged.

For example, send `{"success":true,"rows":[]}` instead of
`{success:true,rows:[]}`. TreeLoader responses must be arrays of node objects.
Invalid tree responses trigger `loadexception` and the failure callback without
also firing `load`. Node construction is validated before immediate children
are appended.

The built column-tree and summary example feeds are converted to JSON through
`overlay/examples/`. PHP endpoints that already use `json_encode` retain their
existing response behavior; other custom servers must adopt strict JSON.

## Reader paths and metadata

JsonReader mappings, `root`, `id`, `totalProperty`, and `successProperty` accept
data property paths rather than JavaScript expressions:

```js
{name: 'name', mapping: 'person.name'}
{name: 'firstItem', mapping: 'items[0].name'}
{name: 'label', mapping: 'details["display.name"]'}
{name: 'label', mapping: "details['display name']"}
```

Numeric array mappings and paths beginning with bracket keys are supported.
Quoted keys accept ordinary JSON escapes; single-quoted keys also support
escaped single quotes. Simple path segments use word characters, `$`, and `-`;
use quoted bracket keys for spaces, dots, and other characters.

Paths reject calls, assignments, expressions, and the keys `__proto__`,
`constructor`, and `prototype`. Lookup uses own properties. Missing or null
intermediate values return `undefined`, allowing field defaults to apply.
Inherited data properties no longer match.

Server-supplied field names and paths are checked before metadata replaces the
reader's configuration or invokes `onMetaChange`. JSON cannot supply converter
functions. Existing application-created field converter functions remain
supported. Put calculated mappings in application-controlled `convert`
functions instead of response metadata.

## Tree UI providers

String `uiProvider` values must name an own entry in the loader's application
registry, and that entry must be a function:

```js
var loader = new Ext.tree.TreeLoader({
    dataUrl: '/nodes.json',
    uiProviders: {custom: MyTreeNodeUI}
});
// Response node: {"text":"Item","leaf":true,"uiProvider":"custom"}
```

Unknown names, inherited registry entries, and JavaScript expressions are
rejected. Application code can still pass a provider constructor directly.
Register aliases for formerly evaluated global constructor names.

## Selectors

Captured selector values are escaped before being inserted into generated
JavaScript strings. Quotes, backslashes, and line separators remain literal
data. Query caches cannot inherit entries from Object.prototype, and pseudo
names resolve only to registered own functions. Ext operators, pseudos,
XPath-like syntax, and application-defined pseudo callbacks remain supported.

Custom matcher code snippets are trusted application code; captured placeholder
values are escaped as contents of double-quoted JavaScript literals. Review
custom matchers that relied on interpolating executable expressions.

## Templates and remaining boundaries

Use `Ext.SafeTemplate` for ordinary data display:

```js
var template = new Ext.SafeTemplate('<p class="greeting">Hello {person.name}</p>');
template.overwrite('app', {person: {name: '<untrusted text>'}});
```

SafeTemplate parses text placeholders without eval or Function. It reads dotted
own-data paths (`{person.name}`, `{items.0.name}`, `{0}`), binds primitive values
as DOM text, and treats missing/null values as empty text. `{{` and `}}` display
literal braces. Getters, object coercion, function values, and prototype paths
are rejected. Its ordinary-data contract covers records/arrays; it is not a
sandbox for arbitrary JavaScript objects or proxies.

Static structure must be application-authored. The supported HTML subset is
text/layout/list/table markup, with static id/class/title/role/lang/dir and table
attributes, plus aria/data attributes. Active elements, URLs, style/event
attributes, attribute placeholders, formatters, and expressions are rejected.
Precompute formatted strings in application code. Loops and conditions are not
part of this initial safe subset.

It is not a drop-in replacement for APIs that require an XTemplate instance.
For a grid, an application renderer can return `safeTemplate.apply({value: v})`
from a template such as `<span>{value}</span>`; that escapes the value while
retaining the trusted static structure. Ordinary widget defaults are unchanged.

`apply`/`applyTemplate` return serialized HTML; `render` returns a fresh fragment.
`append`, `insertFirst`, `insertBefore`, `insertAfter`, and `overwrite` use DOM
operations and accept supported HTML content containers. `set`/`compile`
validate structure. Invalid templates preserve the old definition, and failed
rendering preserves the destination. See [the audit](HTML_PROTOTYPE_AUDIT.md)
for remaining legacy widget rendering and prototype findings.

Template definitions are trusted application code. Compiled formatter arguments
and XTemplate conditions, loops, `exec`, and inline expressions retain their
existing executable behavior. Do not build template definitions from user or
server text. Ordinary string values passed to a fixed template are not
recompiled.

Use `{name:htmlEncode}` for untrusted text in HTML content. Unformatted template
values, tree node markup, component HTML, and explicit script-loading features
remain HTML/code trust boundaries; this change does not sanitize arbitrary HTML
or URLs.

The HTML encoder now escapes both quote characters. This protects quoted
attribute syntax, but it does not validate URLs or sanitize rich markup. JSON
encoding supports hasOwnProperty data fields and null-prototype objects.
Prototype-sensitive copying and collection maps remain audit findings.

JSON decoding and reader path lookup no longer need string compilation.
Selectors, templates, date helpers, and Format helpers still generate code.
The full library still requires a CSP that permits those operations; a complete
nonexecuting renderer/query runtime is outside this patch. The historical docs
UI has its own archived runtime bundle, separate from the generated library
distributions hardened here. Do not use archived bundles as application code.
