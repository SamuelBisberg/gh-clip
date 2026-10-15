<div align="center">

# gh-clip

*A clipboard that follows you from your laptop to your phone - powered by GitHub, no servers*

[![CI](https://github.com/SamuelBisberg/gh-clip/actions/workflows/ci.yml/badge.svg)](https://github.com/SamuelBisberg/gh-clip/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/SamuelBisberg/gh-clip?sort=semver)](https://github.com/SamuelBisberg/gh-clip/releases)

[Install](#install) • [Usage](#usage) • [Configuration](#configuration) • [How it works](#how-it-works)

</div>

A [GitHub CLI](https://cli.github.com) extension that acts as a secure cross-device clipboard. Pipe text in on one machine, copy it out on another - or scan a QR code and tap to copy on your phone. Storage, transfer, and cleanup all live on GitHub: no external server, no database, no background daemon.

## Install

```sh
gh extension install SamuelBisberg/gh-clip
gh clip init
```

Upgrade later with `gh extension upgrade clip`.

`gh clip init` is a one-time setup wizard. It creates your private storage repo, a small public page for the phone handoff, and the hourly cleanup workflow, then asks you to paste a `gist`-scoped token (it opens the right GitHub page for you) so that workflow can delete expired shares.

## Usage

### `echo ... | gh clip`

Saves piped text to a private repo (`gh-clip-storage` by default).

```console
$ echo "DB_PASS=123" | gh clip
✓ Saved clip to you/gh-clip-storage
ℹ Copy it on another device with: gh clip
```

### `gh clip`

With nothing piped in, opens an interactive list of your recent clips.

| Key       | Action                                   |
| --------- | ---------------------------------------- |
| `↑` / `↓` | Move (`j`/`k` also work)                 |
| `enter`   | Copy the clip to the system clipboard    |
| `q`       | Show the clip as a QR code               |
| `d`       | Delete the clip (asks for confirmation)  |
| `esc`     | Quit                                     |

### `... | gh clip --share`

Sends text to your phone. It creates a short-lived **secret Gist** and prints a QR code in the terminal; scanning it opens a full-screen page where tapping anywhere copies the text - no GitHub login on the phone.

```console
$ cat url.txt | gh clip --share
✓ Shared! Scan to open on your phone:
  ▄▄▄▄▄ ▄ ...
ℹ https://you.github.io/gh-clip-viewer/?gist=...
ℹ Expires in 1h0m0s
```

### `gh clip sweep`

Runs the cleanup workflow right now instead of waiting for the hourly schedule.

### `gh clip config`

```sh
gh clip config list
gh clip config get share.ttl
gh clip config set share.ttl 30m
```

## Configuration

Settings live in `~/.config/gh/clip.json` (or your platform's config dir equivalent).

| Key            | Values                    | Default            |                                                       |
| -------------- | ------------------------- | ------------------ | ----------------------------------------------------- |
| `storage.repo` | repo name                 | `gh-clip-storage`  | Private repo clips are stored in.                     |
| `share.ttl`    | duration (`30m`, `2h`)    | `1h`               | How long a `--share` Gist lives before it's swept.    |
| `ui.color`     | `auto`, `always`, `never` | `auto`             | Override terminal color detection.                    |

How long saved clips are kept is set by `clipTTLHours` in `retention.json` in your storage repo (default 168 - one week). Edit the file there to change it.

## How it works

- **Saved clips** are plaintext files in a **private** repo, one per clip, named `<unix-time>-<id>.txt` under `clips/`, written through the GitHub REST contents API. Auth is your existing `gh auth login` session; gh-clip never stores a token itself.
- **Shared clips** are **secret Gists** - unlisted, reachable only by their unguessable ID. The expiry is encoded in the Gist description (`gh-clip-share_<unix-time>`).
- **The phone page** is a static single-file page on GitHub Pages (`<you>.github.io/gh-clip-viewer`). It reads the Gist anonymously and turns the whole screen into a tap-to-copy button, because mobile browsers only allow clipboard writes from a user gesture.
- **Cleanup** is a GitHub Actions workflow in your storage repo that runs hourly: it deletes clips older than the retention TTL (using each file's commit time) and deletes expired Gists using the `GIST_TOKEN` secret.

> [!WARNING]
> Anyone with a share link can read that clip until it expires, and clips in the storage repo are plaintext - don't treat either as a vault for long-lived secrets.

Releases are built by [`cli/gh-extension-precompile`](https://github.com/cli/gh-extension-precompile) whenever a version tag is pushed, producing binaries for Linux, macOS, and Windows - that's what `gh extension install`/`upgrade` fetch.

Want to contribute? See [CONTRIBUTING.md](CONTRIBUTING.md).
