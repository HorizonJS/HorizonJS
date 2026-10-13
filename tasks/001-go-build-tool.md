# 001: Go build tool

Status: complete

Authorized by the user: implement the proposed Go build tool first.

## Scope

- Replace Bash/PowerShell orchestration and C# library generation with one Go CLI.
- Preserve original source, bundle ordering, JSMin behavior, and distribution layout.
- Keep generated documentation in the existing Java/Docker pipeline; provide an explicit bundled-docs option for local builds.
- Record historical missing manifest entries explicitly; fail on new missing inputs.
- Stage and validate output before packaging and publishing it to `build/`.
- Cover failure propagation and output compatibility with tests; update build instructions and CI.

## Validation

- `go test ./...` and `go vet ./...` passed on Windows.
- All 302 retained library files matched the previous C# output byte for byte on Windows; golden tests normalize checkout line endings and cover the main JS/CSS bundles.
- Full default build passed with Docker-generated documentation (190 class pages), three release ZIPs, and the website.
- Full bundled-docs build and tests passed under Linux with LF source files.
- CLI and both compatibility wrappers returned nonzero exit codes for invalid arguments.
- Failure tests covered missing files, unusable targets, malformed JS, corrupt artifacts, Docker build/create/copy/cleanup failures, cancellation, and preservation of previous output.
- The standalone CLI compiled for Windows amd64, Linux amd64, and macOS arm64. macOS runtime execution remains for CI.
- CI definitions were checked with actionlint; the tag workflow's obsolete checkout action was updated during validation.

## Result

`go run ./cmd/horizon build --tag main` is the active build entry point.
`--docs bundled` supports local builds without Docker. The old C# sources and
Dockerfile remain as references. See `BUILD_TOOL.md` for behavior and limitations.

Follow-up: task 004 removes the legacy sources and Docker comparison mode after
the native migrations, retaining saved compatibility baselines.
