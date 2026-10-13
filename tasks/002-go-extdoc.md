# 002: Native Go API documentation generator

Status: complete

Authorized by the user: rewrite Extdoc after the Go build-tool migration.

## Scope

- Port Ext 2 documentation comments, inline links, constructor/parameter tags,
  custom tags, class merging, inheritance, overrides, hidden members, and static members.
- Preserve the existing documentation UI assets, class-page structure, tree data,
  member anchors, and source links with native Go rendering.
- Make native generation the build default, retaining explicit Docker and bundled
  modes as migration references.
- Fail on malformed input and filesystem errors; preserve staged publication.
- Compare all generated classes against saved Java output, test edge cases, and
  verify the documentation UI and cross-platform execution.

## Validation

- Saved the Java reference output and compared all 190 class pages with the
  native output: zero differences after normalizing HTML serialization whitespace.
- Navigation data and icon assignments matched the Java reference exactly.
- Golden tests cover all class pages and check every emitted source link against
  an actual source anchor; 213 escaped source pages are generated.
- `go test ./...` and `go vet ./...` passed on Windows.
- Full generator and build-tool tests passed on Linux using LF source files;
  the complete native build passed with container networking disabled.
- The native CLI compiled for Windows amd64, Linux amd64, and macOS arm64.
- Browser verification passed for class navigation, inherited member rows,
  source navigation, and absence of console errors.
- The final Windows build passed in an isolated destination to avoid concurrent
  browser-module packaging work in the shared checkout.

## Compatibility decisions

- Freeze the observed Java traversal of the original 213 source files, including
  duplicate filenames and order-sensitive class/member behavior. New files are
  scanned deterministically after those entries.
- Preserve legacy tag parsing, incomplete inline links, hidden override behavior,
  and unnamed property rows rather than silently changing the reference docs.
- Normalize comment line endings for cross-platform short-description lengths.
- HTML-escape source code, preserving source anchors and making literal HTML
  readable in source pages.
- Support the repository's Ext template and tags with native Go rendering.
  Arbitrary Extdoc project XML and custom XSLT templates are not interpreted.

## Result

`go run ./cmd/horizon build --tag main` now generates documentation in Go.
The original Java/XSLT code remains untouched as an optional `--docs docker`
reference. See `BUILD_TOOL.md` for modes and limitations.

Follow-up: task 004 removes the legacy sources and Docker comparison mode after
the native migrations, retaining saved compatibility baselines.
