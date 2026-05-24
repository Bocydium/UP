# up

The fast, secure, self-contained package manager for Arch Linux. No subprocess spam. No pacman noise. Just `up`.

## What it does

`up` downloads packages directly from Arch mirrors, extracts them natively, and tracks everything in its own database. For AUR packages, it builds with `makepkg` but caches binaries so you never rebuild the same package twice.

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

## Why it's different

| Feature | yay | up |
|---|---|---|
| Clean output | Noisy pacman spam | Minimal, only what matters |
| Official repo installs | `pacman -S` subprocess | Native HTTP download + extract |
| Security scanning | None | Deep PKGBUILD analysis |
| Binary cache | None | Hash-based, skips rebuilds |
| Dry-run plan | No | `up plan` shows everything before it happens |
| File diff before update | No | `up diff` shows modified/missing files |
| Health score | No | 0-100 score on every AUR package |
| Dependency tree | Flat list | `up tree` with visual tree |
| Backup/restore | No | `up backup` saves package lists |
| Auto-remove deps | Manual | `up remo` cleans everything |
| Flatpak integration | No | Built into `up upda` |

## Architecture

**Official repos (fully native):**
1. Sync repo databases from mirrors over HTTP
2. Search databases for package metadata
3. Download `.pkg.tar.zst` directly from mirror
4. Extract natively with zstd + tar (no `pacman -U`)
5. Track installed files in local database

**AUR (cached builds):**
1. Search AUR RPC API
2. Clone PKGBUILD git repo
3. Run security scan on PKGBUILD
4. Build with `makepkg` (only if not cached)
5. Cache binary by PKGBUILD hash
6. Install from cache on subsequent runs

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
- `makepkg`, `git` (for AUR builds only)
- Optional: `flatpak`

## License

MIT
