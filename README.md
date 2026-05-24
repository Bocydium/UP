# up

The fast, secure package manager for Arch Linux. Like `yay`, but cleaner, safer, and faster.

## What it does

`up` wraps `pacman`, `makepkg`, and the AUR into a single, fast tool with deep security scanning and intelligent caching. No more `clean build? [Y/n]` noise. No more rebuilding the same AUR package every update.

## Commands

```bash
up inst neovim              # Install from official repos or AUR
up inst visual-studio-code-bin --nocheck   # Skip security checks (not recommended)
up remo neovim              # Remove package + unused deps + config files
up upda                     # Update everything: pacman + AUR + flatpak
up search neovim            # Search official repos and AUR
up info neovim              # Show package details
up cache                    # Show cache size
up cache clean              # Clean old cached builds (keeps last 10)
```

## Why it's different

| Feature | yay | up |
|---|---|---|
| Clean TUI | Noisy prompts | Minimal, fast output |
| Security scanning | None | Deep PKGBUILD analysis |
| Binary cache | None | Hash-based, skips rebuilds |
| Auto-remove deps | Manual | `up remo` cleans everything |
| Flatpak integration | No | Built into `up upda` |
| Speed | Good | Parallel downloads, aggressive caching |

## Security

`up` scans every AUR package before building:

- Detects dangerous patterns (`curl | sh`, `eval`, `sudo` in PKGBUILD)
- Verifies checksums are present for network downloads
- Checks GPG key configuration
- Flags low-vote and unmaintained packages
- Flags out-of-date packages

## Binary Cache

`up` caches built AUR packages by hashing the PKGBUILD and `.SRCINFO`. If you install the same package again (or update to a version with the same build files), it skips the build entirely and installs from cache.

Cache location: `~/.cache/up/binaries/`

## Install

```bash
curl -sSL https://raw.githubusercontent.com/aapollo/up/main/install.sh | bash
```

Or manually:
```bash
go build -ldflags="-s -w" ./cmd/up
sudo install -Dm755 up /usr/local/bin/up
```

## Requirements

- Arch Linux (or derivative)
- `pacman`, `makepkg`, `git`, `sudo`
- Optional: `flatpak`

## How it works

1. **Official repos**: `up` checks `pacman -Si` first. If found, installs via `sudo pacman -S`.
2. **AUR**: Searches the AUR RPC API, clones the git repo, runs `makepkg`, installs with `pacman -U`.
3. **Security**: Before building, scans PKGBUILD for dangerous commands and missing checksums.
4. **Cache**: After building, caches the binary by PKGBUILD hash. Future installs skip the build.
5. **Updates**: `up upda` runs `pacman -Syu` (asks for confirmation), then checks all AUR packages for version bumps (with cache), then `flatpak update`.

## License

MIT
