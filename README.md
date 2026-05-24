# up

The fast, secure package manager for Arch Linux. Like `yay`, but cleaner, safer, and actually unique.

## What it does

`up` wraps `pacman`, `makepkg`, and the AUR into a single tool with deep security scanning, intelligent caching, and features no other helper has.

## Commands

```bash
# Install
up inst neovim              # Official repo or AUR (auto-detected)
up inst neovim --plan       # Dry-run: show what would happen

# Remove
up remo neovim              # Deep removal: package + unused deps + configs

# Update
up upda                     # Update everything: pacman + AUR + flatpak

# Search
up search neovim            # Search official repos and AUR with health scores

# Info
up info neovim              # Show details + dependency tree + health score

# Unique features
up plan                     # Dry-run: show all pending updates
up plan neovim              # Dry-run: show what installing neovim would do
up diff neovim              # Show file changes before updating
up tree neovim              # Visual dependency tree
up backup                   # List saved snapshots
up backup create "before update"   # Save package list
up backup restore 20260524  # Restore from snapshot
up cache                    # Show cache size
up cache clean              # Clean old cached builds
```

## Why it's different from yay

| Feature | yay | up |
|---|---|---|
| Clean TUI | Noisy prompts | Minimal, fast output |
| Security scanning | None | Deep PKGBUILD analysis |
| Binary cache | None | Hash-based, skips rebuilds |
| Dry-run plan | No | `up plan` shows everything before it happens |
| File diff before update | No | `up diff` shows modified/missing files |
| Health score | No | 0-100 score on every AUR package |
| Dependency tree | Flat list | `up tree` with visual tree |
| Backup/restore | No | `up backup` saves package lists |
| Auto-remove deps | Manual | `up remo` cleans everything |
| Flatpak integration | No | Built into `up upda` |

## Security

`up` scans every AUR package before building:

- Detects dangerous patterns (`curl | sh`, `eval`, `sudo` in PKGBUILD)
- Verifies checksums are present for network downloads
- Checks GPG key configuration
- Flags low-vote and unmaintained packages
- Flags out-of-date packages

## Health Score

Every AUR package gets a 0-100 health score:

- **Votes** (30 pts): More votes = higher score
- **Maintainer** (25 pts): Orphaned packages get 0
- **Freshness** (25 pts): Out-of-date penalties
- **Popularity** (20 pts): Based on vote count

Shown as a colored bar in `up info` and `up search`.

## Binary Cache

`up` caches built AUR packages by hashing the PKGBUILD and `.SRCINFO`. Reinstalling the same package (or updating to a version with identical build files) skips the build entirely and installs from cache instantly.

- Cache location: `~/.cache/up/binaries/`
- `up cache` shows total size
- `up cache clean` keeps last 10 builds, removes old ones

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

1. **Official repos**: Checks `pacman -Si` first. If found, installs via `sudo pacman -S`.
2. **AUR**: Searches the AUR RPC API, clones the git repo, runs `makepkg`, installs with `pacman -U`.
3. **Security**: Before building, scans PKGBUILD for dangerous commands and missing checksums.
4. **Cache**: After building, caches the binary by PKGBUILD hash. Future installs skip the build.
5. **Updates**: `up upda` runs `pacman -Syu` (asks for confirmation), then checks AUR packages for version bumps (with cache), then `flatpak update`.

## License

MIT
