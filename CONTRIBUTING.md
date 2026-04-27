# Contributing to gensyn

Thank you for your interest in contributing to gensyn. This document covers how to get set up, what the codebase looks like, and the conventions to follow when submitting changes.

---

## Getting started

### Prerequisites

- Gentoo Linux (the program reads live Portage data from `/var/db/repos/gentoo/` and `/var/db/pkg/`)
- Go 1.21 or later
- Fyne system dependencies — see the [README](README.md) for the full list
- `app-admin/sudo`
- `app-portage/gentoolkit` (for command search)

### Clone and run

```bash
git clone https://github.com/Brainbeer/gensyn.git
cd gensyn
LANG=en_US.UTF-8 go run main.go
```

---

## Project structure

```
gensyn/
├── main.go               # Entry point — calls ui.StartUI()
├── models/
│   └── models.go         # Category and Package structs
├── portage/
│   └── portage.go        # Filesystem reads from /var/db/repos and /var/db/pkg
├── ui/
│   ├── window.go         # All UI layout, widgets, and emerge command logic
│   ├── prefs.go          # Preferences struct, load/save, theme implementation
│   ├── bundled.go        # Auto-generated icon resource (do not edit manually)
│   └── icon.png          # Source app icon (256x256)
├── vendor/               # Vendored dependencies
├── go.mod
└── go.sum
```

---

## Ground rules

- **Go step by step.** Make small, focused changes. One feature or fix per pull request.
- **Do not change working code** unless the change is explicitly part of your fix or feature.
- **Full files only** when submitting changes to `window.go` or `prefs.go` — no partial snippets or diffs against intermediate versions.
- **Test on a real Gentoo system.** The program reads live filesystem paths. Changes to portage.go or the tree/package logic must be tested against a real Portage tree.
- **Run with the correct locale.** Always use `LANG=en_US.UTF-8 go run main.go` to avoid encoding issues with emerge output.
- **No new dependencies** without prior discussion. The dependency surface should stay minimal.

---

## Code conventions

### UI (Fyne)

- All UI updates from goroutines must be wrapped in `fyne.Do()`.
- Use the existing `runCommand` / `promptAndRun` functions for any new emerge operations — do not create new command-running patterns.
- New sudo operations should follow the same stdin-only password pattern used throughout the codebase.
- Widget forward references follow the `var foo *widget.Foo` pattern at the top of `StartUI()` — use this for any widget that needs to be referenced in a menu closure before it is assigned.

### Preferences

- Any new preference field goes in the `Prefs` struct in `prefs.go` with a JSON tag and a sensible default in `defaultPrefs()`.
- The Preferences dialog in `window.go` must be updated to include the new field.

### Portage paths

- Portage filesystem paths are hardcoded constants in `portage/portage.go`. If you need to make them configurable, discuss first — this touches both the portage package and the preferences system.

---

## Submitting a pull request

1. Fork the repository and create a branch named after your change: `feature/my-feature` or `fix/my-fix`
2. Make your changes following the conventions above
3. Test on a live Gentoo system
4. Open a pull request with a clear description of what changed and why
5. Reference any related issues in the PR description

---

## Reporting bugs

Open an issue on GitHub and include:

- A description of what you expected to happen and what actually happened
- The output of **Tools → emerge --info** if the issue is emerge-related
- Your Gentoo profile and relevant `make.conf` settings if applicable
- The gensyn version from **About → About gensyn**

---

## License

By contributing to gensyn you agree that your contributions will be licensed under the GNU General Public License v3.0.