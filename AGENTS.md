# AGENTS.md

A collection of small, generic Go packages (Go 1.26) that fill gaps in the
standard library. Module: `github.com/mikeland73/golibs`.

## Commands

Tooling is pinned with devbox. Run everything through it:

```sh
devbox run fmt    # format (gofumpt + goimports via golangci-lint)
devbox run check  # fmt-check + lint + race tests; must pass before committing (CI runs the same)
```

## Rules

- **Standard library only.** No third-party or `golang.org/x` imports, and no
  imports between packages in this repo. Each package must stand alone.
- **No duplication** of the standard library, `golang.org/x/*`, or
  `samber/lo`. If something already exists there, point to it in the docs (see
  the README's "Not included" table) instead of re-implementing it.
- **Go 1.26 max**, since some target environments don't have newer versions.
- **Layout:** one package per top-level directory, usually `<pkg>.go`,
  `<pkg>_test.go` and `example_test.go`.

## API conventions

- Comparators use `cmp func(a, b T) int` (the same shape as `slices.SortFunc`).
  Constructors are `New` for `cmp.Ordered` types and `NewFunc` for a custom
  comparator, unless a package documents otherwise.
- Containers provide `Len`, `IsEmpty`, `Clear`, `All()` (an iterator),
  `ToSlice` and `String` where they make sense.
- `Pop`/`Peek` panic with a clear message when the container is empty;
  `TryPop`/`TryPeek` return `(v, ok)` instead.
- Make the zero value usable where that's cheap, and document it.
- Every exported identifier has a doc comment. Data-structure docs state the
  Big-O cost of each operation. Wrap comments at 80 columns.
- Tests use only the stdlib `testing` package (no testify). Include edge cases,
  randomized checks against a simple reference implementation where useful,
  and runnable `Example` functions. Concurrency code must pass `-race`; use
  `testing/synctest` instead of sleeps.
