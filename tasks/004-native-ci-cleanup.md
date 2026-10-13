# 004: Native CI and legacy tool removal

Status: complete

Authorized by the user: update GitHub Actions to use the Go tools and remove
the old tool implementations.

## Scope

- Use native Go library and documentation generation in Pages, releases,
  cross-platform checks, and the browser module build.
- Replace the Docker-based release action with GitHub CLI and restrict token
  permissions to each job's needs; deploy Pages only after a successful build.
- Remove C# JSBuild, Java/Ant Extdoc, their Dockerfiles, and obsolete helpers.
- Preserve documentation UI assets and their license in `assets/docs/`, with
  saved Java/C# compatibility baselines retained in the Go tests.
- Remove the Docker documentation mode and external-command plumbing; preserve
  the explicit bundled snapshot option.
- Update documentation and verify workflows and builds after deletion.

## Validation

- Hash-verified all 296 relocated documentation assets; preserved the original
  license and included it in generated documentation output.
- Confirmed all six obsolete tool/source/container/helper paths are absent.
- `go test ./...` and `go vet ./...` passed on Windows, including saved bundle
  and 190-class documentation baselines, invalid-mode handling, and input safety.
- All four workflows passed actionlint.
- Full native website and versioned release builds passed; the latter produced
  all three ZIPs. The standalone Windows CLI was rebuilt, passed a bundled-docs
  build, and returned failure for the removed Docker mode.
- `npm run test:browser` passed Chromium checks for native imports, late imports,
  classic scripts, Vite development, production under `/demo/`, the actual npm
  tarball, and CSS asset references using the new native build script.
- Firefox checks could not launch locally. A direct Windows process launch
  confirmed the cached Playwright Firefox executable has an incorrect
  side-by-side configuration. The Ubuntu CI workflow installs its own browsers.
- `git diff --check` passed. Original `extjs-2.0.2/` inputs remain unchanged.
- Workflows were validated locally; GitHub deployment and release execution
  require these changes to be pushed.
