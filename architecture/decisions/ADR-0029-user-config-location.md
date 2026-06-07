# ADR-0029 — Unified per-user config location (`~/.oblikovati`)

**Status:** accepted (user decision, 2026-06) · **Relates to:** [ADR-0020](ADR-0020-yaml-git-friendly-document-format.md)
(documents), [ADR-0021](ADR-0021-ui-theming-semantic-tokens.md) (themes)

## Context

Global, user-bound settings are persisted by several independent stores — themes
(`theme.Store`), per-document view/camera state (`persistence/viewstate`), window placement
(`head/internal/windowstate`), and global UI preferences (`persistence/userprefs`). Each
called `os.UserConfigDir()` itself, so the files scattered to the OS-native roots:
`~/.config/oblikovati` (Linux), `~/Library/Application Support/oblikovati` (macOS), and
`%AppData%\oblikovati` (Windows). That made a user's settings inconsistent across platforms
and hard to find on macOS, and risked subtle drift as each store re-derived the path.

## Decision

One source of truth, the **`oblikovati/userconfig`** package, resolves the per-user config
directory; every store routes through `userconfig.Dir()` / `userconfig.File(name)` and none
calls `os.UserConfigDir` directly (the build is grepped to keep it that way):

- **Linux & macOS:** `~/.oblikovati` — a single, discoverable dotfolder in `$HOME`, the
  same on both so a user's settings live together.
- **Windows:** `%AppData%\oblikovati` (the platform-native roaming AppData location).

Files under `<userconfig>`:

| File / dir              | Owner                         | Contents                                            |
|-------------------------|-------------------------------|-----------------------------------------------------|
| `themes/*.yaml`         | `theme.Store`                 | custom themes (one per file)                        |
| `preferences.yaml`      | `theme.Store`                 | the selected theme                                  |
| `ui-preferences.yaml`   | `persistence/userprefs`       | global UI prefs (e.g. ViewCube compass visibility)  |
| `view-state.yaml`       | `persistence/viewstate`       | per-document cameras/views/layout, keyed by path    |
| `window.yaml`           | `head/internal/windowstate`   | last window position/size/maximized                 |

The global UI prefs file is named **`ui-preferences.yaml`**, not `preferences.yaml`, because
the theme store already owns `preferences.yaml` in the same directory; sharing the name would
have made the two clobber each other.

## Consequences

- All global/user-bound settings are co-located and discoverable; new global stores join by
  calling `userconfig.File("…")` — no per-store path logic.
- **macOS deviates** from the native `~/Library/Application Support` convention. Accepted: a
  single `~/.oblikovati` across the Unix-likes wins on consistency and discoverability for a
  CAD tool whose users expect a visible dotfolder.
- **Per-document** content is explicitly NOT here: `.obk` documents are the document format
  (ADR-0020) and live wherever the user saves them. `userconfig` is strictly global/user
  state, kept out of the document so a setting never dirties a document in git.
- **Migration:** changing the location orphans any files previously written under the
  OS-native roots. The app is alpha and these files are regenerable, so no migration is
  performed (a user may copy the old directory's contents into `~/.oblikovati` once).
