# System Tray Library Alternatives

Research and evaluation of replacements for `github.com/getlantern/systray`
(currently pinned at `v1.2.2` in `go.mod`). No code changes are made by this
document — it is a decision record.

## Current Usage Summary

The application uses the system tray library in exactly two files:

- `src/internal/tray/runner.go` — the tray UI and update logic
- `src/cmd/run_tray.go` — process startup and signal-driven shutdown

### APIs in use

All ten `getlantern/systray` APIs touched by the codebase:

| # | API | Type | Used in |
|---|-----|------|---------|
| 1 | `systray.Run(onReady, onExit func())` | package func | `runner.go:36` |
| 2 | `systray.Quit()` | package func | `runner.go:101`, `run_tray.go:36` |
| 3 | `systray.SetTitle(string)` | package func | `runner.go:55,110,116,127,145,172,192,194` |
| 4 | `systray.SetTooltip(string)` | package func | `runner.go:56` |
| 5 | `systray.AddMenuItem(title, tooltip string) *MenuItem` | package func | `runner.go:60,64,66` |
| 6 | `systray.AddSeparator()` | package func | `runner.go:63,65` |
| 7 | `MenuItem.SetTitle(string)` | method | `runner.go:160` |
| 8 | `MenuItem.Hide()` | method | `runner.go:158,163` |
| 9 | `MenuItem.Show()` | method | `runner.go:159` |
| 10 | `MenuItem.ClickedCh` (`chan struct{}`) | field | `runner.go:98,99` |

The `*systray.MenuItem` type is also stored as a slice field on the `Runner`
struct (`runner.go:18,28`).

### Explicitly unused APIs

- `systray.SetIcon()` / `MenuItem.SetIcon()` — the project ships **no icon
  assets**; the tray presents a text-only title (`SetTitle`). Any candidate's
  icon API is irrelevant.
- `systray.SetTemplateIcon()`, `MenuItem.AddSubMenuItem()`,
  `MenuItem.Checkbox`/`Check()`/`Uncheck()`, `systray.ResetMenu()` — not used.
- `RunWithExternalLoop()` — not used (no external event loop is integrated).

### Platform constraints

- **Release targets are macOS-only**: `.goreleaser.yaml` builds `darwin/amd64`
  and `darwin/arm64`, combined into a universal binary. No Linux or Windows
  release is produced.
- **CGO is required** for the macOS build: `.goreleaser.yaml` sets
  `CGO_ENABLED=1` with `-mmacosx-version-min=12.0` for both `CGO_CFLAGS` and
  `CGO_LDFLAGS`. systray on macOS binds to the Cocoa status-bar APIs and cannot
  build without CGO.
- The Linux release is **intentionally omitted** (see comment in
  `.goreleaser.yaml`): `getlantern/systray` on Linux needs CGO plus
  `libayatana-appindicator3-dev` and `pkg-config`.

### Build tag constraints

- `src/cmd/run_tray.go` carries `//go:build !nogui` — it is the GUI entry point.
- `src/cmd/run_notray.go` carries `//go:build nogui` — an empty stub enabling
  **headless builds** that exclude the systray dependency entirely.
- Any replacement must preserve this split: the `nogui` build must continue to
  compile without pulling in the tray library or CGO.

### Requirements checklist

A replacement library must satisfy all of the following:

- [ ] Provides `Run(onReady, onExit func())` with identical signature
- [ ] Provides `Quit()`
- [ ] Provides `SetTitle(string)`
- [ ] Provides `SetTooltip(string)`
- [ ] Provides `AddMenuItem(title, tooltip string) *MenuItem`
- [ ] Provides `AddSeparator()`
- [ ] `MenuItem` exposes `SetTitle(string)`
- [ ] `MenuItem` exposes `Hide()`
- [ ] `MenuItem` exposes `Show()`
- [ ] `MenuItem` exposes a `ClickedCh` channel field
- [ ] Builds on `darwin/amd64` and `darwin/arm64` (mandatory)
- [ ] Compatible with `CGO_ENABLED=1` and `-mmacosx-version-min=12.0`
- [ ] Does not break the `nogui` / `!nogui` build-tag separation
- [ ] No change required to `.goreleaser.yaml` build matrix

## Evaluation Criteria

Candidates are judged against five criteria derived from the checklist above:

1. **API compatibility** — must expose all 10 APIs with identical signatures so
   the migration is an import-path swap with zero call-site edits.
2. **Platform support** — macOS support is **mandatory** and must work under
   CGO. Linux/Windows support is a non-blocking nice-to-have given the
   macOS-only release matrix.
3. **Maintenance status** — actively maintained, recent releases, responsive
   issue tracker. `getlantern/systray` is effectively unmaintained.
4. **Dependency footprint** — fewer transitive dependencies is better. The
   current library drags in 8+ indirect `getlantern/*` modules (`context`,
   `errors`, `golog`, `hex`, `hidden`, `ops`, plus `oxtoacart/bpool`,
   `go-stack/stack`) that exist only to serve `getlantern/systray`.
5. **CGO requirements** — must not introduce CGO needs beyond the existing
   macOS `CGO_ENABLED=1` build; must not regress the headless `nogui` build.

## Alternatives Evaluated

### `fyne-io/systray`

Repository: <https://github.com/fyne-io/systray>

- **API surface** — a maintained fork of `getlantern/systray` that keeps the
  same core API: `Run`, `Quit`, `SetTitle`, `SetTooltip`, `AddMenuItem`,
  `AddSeparator`, and the `MenuItem` methods/`ClickedCh` field are unchanged.
  All 10 required APIs are present with identical signatures.
- **`RunWithExternalLoop`** — adds `RunWithExternalLoop` for embedding the tray
  in an existing event loop. **Not needed** here; it is purely additive and
  does not affect the existing `Run` entry point.
- **Linux** — removes the GTK/`libappindicator` dependency, driving the Linux
  tray via DBus / the StatusNotifier specification instead. Linux support is
  documented by the project as **"work in progress"** — **non-blocking** for
  this macOS-only project.
- **Maintenance** — maintained by the Fyne project team as part of the wider
  Fyne GUI toolkit ecosystem; actively used and updated.
- **Dependencies** — drops the 8+ indirect `getlantern/*` modules.

### `energye/systray`

Repository: <https://github.com/energye/systray>

- **API surface** — core API compatible with `getlantern/systray`: `Run`,
  `Quit`, `SetTitle`, `SetTooltip`, `AddMenuItem`, `AddSeparator`, and the
  `MenuItem` methods/`ClickedCh` field. All 10 required APIs present.
- **Extra callbacks** — adds mouse-event callbacks `SetOnClick`, `SetOnDClick`,
  and `SetOnRClick`. Additive and unused by this project.
- **Linux/BSD** — uses a DBus-based approach for Linux and BSD.
- **Maintenance** — actively released; `v1.0.3` published in early 2025.
- **Dependencies** — also drops the indirect `getlantern/*` modules.

## Comparison Matrix

| Criterion | `getlantern/systray` (current) | `fyne-io/systray` | `energye/systray` |
|-----------|-------------------------------|-------------------|-------------------|
| API compatibility (10 APIs) | baseline | full — identical signatures | full — identical signatures |
| Maintenance status | effectively unmaintained | active (Fyne team) | active (`v1.0.3`, early 2025) |
| Platform support — macOS | yes (CGO) | yes (CGO) | yes (CGO) |
| Platform support — Linux | GTK + `libappindicator` + CGO | DBus / StatusNotifier, "WIP" | DBus-based |
| Dependency count | 8+ indirect `getlantern/*` | removes 8+ indirect deps | removes 8+ indirect deps |
| Migration effort | n/a | import-path swap only | import-path swap only |
| CGO / build tags | `CGO_ENABLED=1`, `nogui` split | unchanged | unchanged |

### Migration effort (both candidates)

For either fork the migration is **an import-path swap only**:

- `github.com/getlantern/systray` → `github.com/fyne-io/systray` *or*
  `github.com/energye/systray`
- `func Run(onReady, onExit func())` and the `MenuItem` struct signature are
  **identical** across all three libraries — no call-site edits in
  `runner.go` or `run_tray.go`.
- Both forks remove the 8+ indirect `getlantern/*` dependencies from `go.sum`.
- **No changes** expected to CGO settings, the `nogui` / `!nogui` build tags,
  or `.goreleaser.yaml`.

## Recommendation

**Primary: adopt `fyne-io/systray`.**

Rationale:

- **Drop-in API parity** — it is a direct fork of `getlantern/systray` and
  preserves every one of the 10 APIs this codebase uses, making the change a
  pure import-path swap with zero call-site edits.
- **Credible long-term maintenance** — it is maintained by the Fyne project
  team as an integral part of an actively developed GUI toolkit, which is a
  stronger maintenance guarantee than a standalone fork.
- **Smaller dependency footprint** — removes the 8+ unmaintained indirect
  `getlantern/*` modules.
- **Risk-free on the only platform that ships** — macOS support is solid under
  CGO; the "work in progress" Linux support is irrelevant because no Linux
  release is produced.

### Secondary option

**`energye/systray`** — a valid fallback if `fyne-io/systray` later proves
unsuitable. It is equally API-compatible and actively released (`v1.0.3`).

Conditions under which the secondary becomes preferable:

- If the project later wants the additional mouse-event callbacks
  (`SetOnClick`, `SetOnDClick`, `SetOnRClick`) for richer tray interaction.
- If `fyne-io/systray` maintenance stalls or a macOS regression appears there.

## Links

- Current library — `getlantern/systray`: <https://github.com/getlantern/systray>
- Candidate 1 — `fyne-io/systray`: <https://github.com/fyne-io/systray>
- Candidate 2 — `energye/systray`: <https://github.com/energye/systray>
