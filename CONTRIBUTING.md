# Contributing to gh-clip

Thanks for taking the time to contribute!

## Prerequisites

- [Go](https://go.dev) 1.25+ (see `go.mod`)
- [GitHub CLI](https://cli.github.com) (`gh`), to run the extension locally

## Getting started

```sh
git clone https://github.com/SamuelBisberg/gh-clip.git
cd gh-clip
go build ./...
```

Install it as a local `gh` extension to try your changes end-to-end:

```sh
gh extension install .
gh clip init
echo "hello" | gh clip
```

> [!TIP]
> After changing code, `gh extension remove clip && gh extension install .` picks up a fresh build.

## Making changes

```sh
go build ./...
go vet ./...
gofmt -l .          # should print nothing
go test -race ./...
```

All four must pass before opening a PR - CI runs the same checks.

### Project layout

| Package             | Responsibility                                                                    |
| ------------------- | --------------------------------------------------------------------------------- |
| `cmd/`              | The `gh clip` command tree ([cobra](https://github.com/spf13/cobra))              |
| `pkg/store`         | Clips as files in the private storage repo (REST contents API)                    |
| `pkg/gist`          | Expiring secret Gists and viewer URLs for phone sharing                           |
| `pkg/scaffold`      | `init` infrastructure: repos, retention policy, cleanup workflow, viewer page     |
| `pkg/gh`            | Thin `REST` interface over go-gh, plus API helpers                                |
| `pkg/config`        | Persisted settings (`~/.config/gh/clip.json`)                                     |
| `pkg/tui`           | Terminal styling, spinners, the bubbletea clip list, QR rendering, prompts        |
| `internal/ghtest`   | Scriptable fake `gh.REST` for tests                                               |

The cleanup workflow (`pkg/scaffold/files/cleanup.yml`) and the mobile viewer (`pkg/scaffold/files/viewer.html`) are embedded into the binary and pushed by `gh clip init`.

### Tests

Everything that talks to GitHub goes through the `gh.REST` interface, so tests use `internal/ghtest.Fake` - no network access and no real account needed. The `cmd` tests swap in the fake, plus fakes for the clipboard, browser, and token prompt, through package-level seams in `cmd/common.go` and `cmd/init.go`.

> [!WARNING]
> `gh clip init` creates real repositories on your account. Try it on a throwaway account, or set `storage.repo` to a scratch name first (`gh clip config set storage.repo my-scratch-clips`).

## Submitting a pull request

1. Fork the repo and create a branch off `main`.
2. Make your change, with tests.
3. Run the checks above.
4. Open a PR - the template will guide you through the rest.

## Reporting bugs / requesting features

Please use the issue templates; they ask for just enough context to act on a report quickly.
