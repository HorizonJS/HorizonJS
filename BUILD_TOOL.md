# Go build tool

The build tool lives in `cmd/horizon` and `internal/buildtool`. It uses only the
Go standard library. Go 1.22 or newer builds the tool; compiled binaries do not
require Go, Java, Ant, Docker, .NET, Bash, PowerShell, or an external ZIP utility
at runtime.

```sh
go run ./cmd/horizon build --tag main
go run ./cmd/horizon build --tag main --docs bundled
go build -o horizon ./cmd/horizon
```

On Windows, use `go build -o horizon.exe ./cmd/horizon`.

## Documentation modes

- `--docs go` (default) generates documentation natively from staged source,
  using the preserved documentation UI assets. No external runtime is needed.
- `--docs bundled` copies the original checked-in API documentation. This is
  useful for reproducing the upstream snapshot; these docs do not
  reflect later source patches.

Both modes produce the library, docs, examples, and three ZIPs. The `main` tag
also produces the website. `--root` selects a checkout explicitly; otherwise the
tool searches upward from the working directory. `--out` selects an output
directory, relative to the working directory if it is not absolute. Output paths
that overlap repository inputs are rejected.

`build.sh` and `build.ps1` are compatibility entry points that forward to Go and
propagate failures. For example: `./build.sh main --docs bundled` or
`./build.ps1 main --docs bundled`. The C# and Java/Ant sources, container recipes,
and obsolete helper scripts have been removed. Saved compatibility baselines
remain in the Go tests.

`--docs docker` is no longer supported. Use native generation or the bundled
snapshot option.

## GitHub Actions

The main workflow tests and vets the Go tools, builds with `--docs go`, and
uploads `build/website` before a separate Pages deployment job runs. Only the
deployment job has Pages write and OpenID permissions.

Tag builds run the same native checks and publish the three release ZIPs with
GitHub CLI. Reruns replace matching release assets while preserving the release
metadata. The release job has contents write permission.

The build-tool matrix runs native tests, vet, and a complete build on Linux,
Windows, and macOS. Browser checks use `npm run build`, which also selects native
Go documentation, and test Chromium and Firefox against the resulting package.

## Failure and publication behavior

Each build uses a fresh staging directory beside the destination. Required
manifest inputs, target includes, parsing, filesystem operations, and ZIP
writes must succeed. The tool validates nonempty library,
adapter, CSS, image, documentation, and example artifacts after applying overlays
and before packaging. Generated docs must include representative class pages,
which catches an empty documentation run.

Only completed output replaces the destination. The old directory is moved to
a temporary backup, the staged build is moved into place, and the backup is
removed. If the second move fails, the tool attempts to restore the old build;
if restoration also fails, the error identifies its retained backup path.
This replacement uses two renames, so readers may briefly see no destination.
Do not run concurrent builds to the same destination.

## Historical manifest entries

The preserved source manifest lists 182 absent files and the resource manifest
lists 344. `internal/buildtool/legacy-missing.json` freezes these exact names.
They are skipped only for their corresponding original manifest; newly missing
files fail. This list is checked in and embedded in the tool, never generated
from the current filesystem during a build.

Four of these names are also stale target includes:

- Everything: `widgets/layout/AbsoluteFormLayout.js`
- Grid: `widgets/grid/Grid.js`, `widgets/grid/AbstractGridView.js`
- Dialog: `widgets/BasicDialog.js`

Skipping them preserves the previous builder's output. It does not restore
those historical classes or assert that the partial package targets work.
Package targets are generated for compatibility and removed from the final
distribution, matching the previous pipeline. A target with no usable includes
always fails, including when all its missing entries are in the compatibility
list. Changes to the list should be reviewed as explicit compatibility changes.

## Compatibility and checks

JSMin is a direct port of the existing 2003 algorithm, with its original license
notice retained. The debug/CSS comment removal behavior is also preserved.
Unpatched builder output retains the original bundle bytes. Production builds
apply deliberate security changes before minification; their output differs.

```sh
go test ./...
go vet ./...
```

Unpatched builder tests compare line-ending-normalized SHA-256 values against output captured
from the C# builder for the four main bundles, standalone adapter, and two
retained CSS files. They also
exercise missing inputs, malformed JavaScript, empty targets, invalid arguments,
rejected legacy documentation modes, artifact validation, cancellation, ZIP contents, and preservation
of a previous build. The original baseline hashes remain unchanged; production
patches are checked separately through build failure tests and packaged browser
security regressions. CI runs on Windows, Linux, and macOS.

## Shared source patches

`overlay/source/patches.json` contains security patches applied while reading
source inputs, before JSMin, debug concatenation, and source staging. The module
is assembled from that staged source, so both distributions receive the fixes.
Generated source documentation also reflects the patched implementation.

Each patch names a source-relative `File`, exact LF-normalized `Before` text,
and an expected `Count`. Use `After` for an inline replacement or `AfterFile`
for a readable JavaScript replacement relative to the patch manifest. Invalid
patches, missing replacement files, mismatched counts, and unused patches fail
the staged build and preserve existing output. Review the snippets in
`overlay/source/` alongside the patch manifest. Upstream files are not edited.

Module-specific strict-mode patches in `overlay/module/patches.json` run afterward.
Their replacements use the same count checks. Changes to shared patches must
also keep module patch expectations valid. See [RUNTIME_SECURITY.md](RUNTIME_SECURITY.md)
for the input contracts and remaining trusted-template/CSP boundaries.

The build tool creates the same website template as before. Its Markdown viewer
still loads Marked from a CDN; this migration does not make that viewer offline.

## Browser module distribution

Every build also creates `build/npm/index.mjs` and a package manifest with
browser, CSS, theme, resource, and legacy exports. The module concatenates the
standalone adapter and Everything targets in manifest order before source
cleanup. Reviewed, counted replacements from `overlay/module/patches.json`
correct strict-mode initialization, namespaces, selectors, animations, event
registration, date pickers, and drag/drop local variables. Missing source,
unmatched replacements, and unused patches fail the staged build.

`overlay/npm` supplies package metadata and usage documentation. The completed
classic distribution and upstream LICENSE.txt are copied into the package.
CSS keeps its image-relative layout; the module resolves the blank image with
`new URL('./resources/images/default/s.gif', import.meta.url)`. The package
version follows semantic-version release tags, falling back to the template's
development version for other builds. The build does not publish to npm.

`npm run test:browser` builds and installs an actual tarball in a temporary
application, validates stylesheet image references, and runs Chromium tests
against native imports, late imports, classic scripts, Vite development, and
minified Vite production under `/demo/`. `npm run test:browser:firefox` runs the
same checks in Firefox. Install the respective Playwright browsers first.
Legacy runtime code generation remains and bundlers may emit direct-eval and
bundle-size warnings; this is not a per-widget tree-shakeable distribution.

## Native documentation compatibility

`internal/extdoc/` replaces Java parsing and XSLT rendering with Go models and
an embedded class-page template. Runtime resources still come from the original
`assets/docs/` directory, including their original license notice. The original
Java sources and XSLT templates have been removed; custom XSLT templates and
arbitrary Extdoc project XML files are not interpreted by the native renderer.

The embedded `legacy-source-order.json` records the 213-file traversal observed
in the Java reference build. This stabilizes duplicate source filenames,
class merging, and order-sensitive inheritance behavior across platforms. New
source files are scanned after those entries in sorted order. Comment line
endings are normalized for the original UTF-16 description truncation rule.

The port retains legacy behavior for literal `@` tokens, incomplete inline
links, unnamed property rows, and hidden overrides. It does not reinterpret
these comments as modern JSDoc. Source pages now HTML-escape JavaScript so tags
inside code remain visible; all emitted source links are checked against anchors.

Golden tests compare member IDs, hyperlinks, row classifications, and visible
text for every one of the Java reference's 190 class pages. Navigation and
icons are also compared against the saved reference. Parser tests cover
constructors, aliases, inheritance, static/private/hidden members, class merging,
custom tags, duplicates, malformed input, and cycles. The default build runs no
external commands.

For a full HTML token comparison against saved Java docs, use the optional
development-only checker (Python is not a build dependency):

```sh
python scripts/compare-docs.py <java-docs-directory> <go-docs-directory>
```
