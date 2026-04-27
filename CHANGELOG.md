# Changelog

All notable changes to gensyn will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [1.0.0] - 2026-04-27

### Added

- Four-panel layout: category tree, package panel, description panel, output terminal
- Category tree reading from `/var/db/repos/gentoo/`
- Installed package detection via `/var/db/pkg/` with **✓** bold indicator in tree
- Installed cache with automatic refresh after install, uninstall, depclean, and world update
- Package details panel showing installed version and ebuild file list
- Package description parsed from `DESCRIPTION=` field in the first ebuild found
- Package search — case-insensitive substring match across all categories
- Command search — uses `equery belongs` to find the package owning a file or command
- Search results dialog for multiple matches; direct navigation for single match
- `emerge` operations via toolbar radio buttons:
  - `No Flag` — install selected package
  - `-p (pretend)` — dry run, no sudo required
  - `-f (fetch)` — fetch sources only
  - `-uvNDU (Update)` — update world
  - `-C (Uninstall)` — unmerge selected package
  - `--depclean` — clean orphaned dependencies
  - `Custom` — free-form flags entry
- **Sync** button running `emerge --sync`
- **Install** button wired to all emerge operations
- sudo password dialog with Enter key support for all privileged operations
- Output terminal streaming stdout and stderr line by line with auto-scroll
- ANSI escape code stripping in terminal output
- Bottom toolbar for writing portage config entries:
  - Supports `package.use`, `package.mask`, `package.accept_keywords`
  - Writes to a per-package file inside the selected directory
  - Uses `sudo sh -c printf` to avoid password/content bleed on stdin
- **View** menu for browsing and opening portage config files:
  - `package.use`, `package.mask`, `package.accept_keywords` — directory listing
  - `make.conf` — single file
  - View contents in a scrollable dialog
  - Edit in configured text editor via sudo
- **Tools** menu with `emerge --info` output displayed in a dialog
- **Edit → Preferences** with persistent settings saved to `~/.config/gensyn/prefs.json`:
  - Dark/light theme
  - Font size (9–14pt)
  - Default emerge operation
  - Clear output on new command toggle
  - Text editor selection (Mousepad, Pluma, Kwrite, Kate, Gedit, Geany, Xed, Featherpad, Sublime Text, VSCode, Atom, Custom)
  - Custom editor path
- **About** dialog with version, author, license, and project URL
- App icon bundled via `fyne bundle`
- `forcedTheme` implementation for reliable dark/light switching and font size control in Fyne v2