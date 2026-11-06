# plumcake

AppImages promise single-file simplicity, but leave loose binaries in `~/Downloads`, missing application icons, and untracked updates.

**plumcake** is a lightweight, keyboard-driven TUI that adopts standalone AppImages, extracts icons, integrates desktop launchers, and manages sandboxing.

---

## Features

- **Desktop integration** — Extracts embedded icons and writes clean XDG desktop entries to `~/.local/share/applications`.
- **Portable & sandboxed** — Toggle per-app `.home`/`.config` directory isolation or sandbox execution via Bubblewrap (`bwrap`) and Firejail.
- **System Doctor** — Audits storage permissions, `$PATH`, desktop file databases, and icon caches with one-key auto-repairs.
- **Safe disposal** — Moves removed applications to managed quarantine with 30-day recovery.
---

## Install

```bash
git clone https://github.com/xibhi/plumcake.git
cd plumcake
make
make install # installs to ~/.local/bin/plumcake
```

---

## Keybindings

| Key | Action |
| :--- | :--- |
| `1` / `2` / `3` | Switch views (*Add*, *Apps*, *Settings*) |
| `/` | Filter list or focus URL/path input |
| `Enter` | Adopt AppImage or run doctor repair |
| `l` | Launch application detached |
| `p` | Toggle portable directory isolation |
| `r` | Rename desktop entry |
| `i` | Replace icon (PNG / SVG) |
| `u` | Move application to trash |
| `?` | Toggle cheat sheet overlay |
| `q` | Quit |

---

## FAQ

**Where do files live?**  
Binaries sit in `~/.local/share/plumcake/apps/`. Desktop shortcuts and extracted icons go to standard XDG paths (`~/.local/share/applications/` and `~/.local/share/icons/`).

**How does portable mode work?**  
Creates adjacent `<AppImage>.home` and `<AppImage>.config` directories. AppImage redirects user data there instead of polluting `$HOME`.

**How does sandboxing work?**  
Wraps launch command with `bwrap` (Bubblewrap) or `firejail`, restricting access to filesystem trees and toggling network access.

**Does plumcake need root privileges?**  
No. Everything runs inside user space under `$HOME` and `$XDG_*` locations.

**What happens on uninstall?**  
Applications move to quarantine trash with a 30-day retention countdown before hard deletion. Expired items purge automatically on startup.

---

## License

This project is licensed under the [MIT License](LICENSE).
