# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go anti-detection headless-Chrome automation library (port of `puppeteer-real-browser`). It launches a real Chrome process and drives it over CDP, injecting stealth + fingerprint scripts so services like Cloudflare/Turnstile don't flag it as a bot. Import path is `github.com/r0vx/puppeteer-real-browser-go` (per `go.mod`).

## Build / test / run

- **`go build ./...` fails by design** — `cmd/example/` holds many files each with its own `func main()` in one `package main`. (`cmd/test_ext/` is a separate nested module with its own `go.mod`, so `./...` skips it.) Build/vet the library only:
  ```bash
  go build ./pkg/... ./internal/...
  go vet ./pkg/... ./internal/...
  ```
- **Run a demo individually** (never `go run ./cmd/example`):
  ```bash
  go run cmd/example/simple_demo.go
  go run cmd/example/cloudflare_demo.go
  ```
- **Tests are integration tests that launch real Chrome** (`pkg/browser` only). Chrome/Chromium must be installed.
  ```bash
  go test ./pkg/browser/                    # full, launches Chrome
  go test -short ./pkg/browser/             # skips pool integration tests
  go test -run TestConnect ./pkg/browser/   # single test
  ```
- **README is partly stale**: it references a `Makefile` (none exists) and Go 1.23; actual toolchain is Go 1.25. Ignore its `make` targets.

## Architecture

Public entry point: `browser.Connect(ctx, opts *ConnectOptions) (*BrowserInstance, error)` (`pkg/browser/browser.go`). Flow:

1. **Launch** (`launcher.go`) — finds Chrome, picks a free port, builds flags (`internal/config`, `internal/utils`), starts the process, sets up Xvfb on Linux.
2. **Connect** — branches on `opts.UseChromedp` (since 2026-10; `UseCustomCDP` is deprecated and ignored):
   - **false (default)** → `CustomCDPConnector` (`cdp_custom.go`), page type `CustomCDPPage`: a browser-level CDP connection (`cdp_conn.go`) whose target manager applies the identity to every page / cross-site iframe / worker / popup before it runs; never calls `Runtime.enable`. `CreateBrowserContext().NewPage()` opens CustomCDP tabs on the same connection (`openTab`; such tabs close only themselves).
   - **true** → legacy `CDPConnector` (`connector.go`), page type `CDPPage`, built on `chromedp`: identity on the main page only. Only for code that calls chromedp directly via `GetContext()`.
3. Both page types implement the `Page` interface (`types.go`). Richer selector/cookie/storage methods live on the `PageWithSelector` interface — reach them via type assertion (`page.(*CDPPage)` / `*CustomCDPPage`).

**Identity (anti-detection)** — design: `docs/superpowers/specs/2026-09-30-coherent-identity-design.md`, applied before any page script runs:
- `identity.go` derives an `Identity` from the account fingerprint (OS comes from the bound UA; Chrome version always follows the real browser) or, with no fingerprint, from the real browser minus headless traces. Real-browser values are read on the startup tab (`identity_apply.go`: `readHostInfo`, `browserHostInfo`) — a CDP-created `about:blank` is not a secure context and has no `navigator.userAgentData`.
- CustomCDP path: `target_manager.go` connects at browser level (flatten sessions over `cdp_conn.go`), auto-attaches every page / cross-site iframe / worker / shared worker / popup with `waitForDebuggerOnStart`, applies the identity through CDP Emulation (`applyIdentity`) plus a minimal JS layer disguised as native code (`identity_script.go`), then resumes it. The chromedp path applies the identity to its main page only.
- If `opts.FingerprintUserID` is set → `UserFingerprintManager` (`user_fingerprint_manager.go`) loads (normalizing and rewriting old files via `FingerprintConfig.Normalize`; a corrupt file is an error) or generates a per-user `FingerprintConfig` (`fingerprint_config.go`, pools in `identity_pools.go`) and **persists it as JSON in `opts.FingerprintDir` (default `./fingerprints/`)**, so the same user ID always gets the same identity across runs.
- Legacy scripts, deprecated and no longer injected by `Connect`: `stealth.go`, `fingerprint_injector.go`, `enhanced_audio_webgl_injector.go`, `timestamp_fingerprint_injector.go`, `script_cache.go`; also `advanced_fingerprint_manager.go`, `runtime_bypass.go`, `rebrowser_patches.go` (`GetSimpleStealthScript` is still used by the chromedp `TargetHandler`).
- Linux production-like tests: `scripts/test-linux-chrome.sh` (amd64 Google Chrome in Docker); deployment notes in `docs/deploy/`.

**Other subsystems (all in `pkg/browser`):**
- `pool.go` — `BrowserPool` reuses instances (has a fast-path health check on `lastUsed`); `account_manager.go` — isolated persistent profiles per account (`ProfileName` / `PersistProfile`).
- `mouse.go` + `pkg/page/controller.go` — ghost-cursor-style Bézier `RealClick`; `wait.go` — smart waits/retries.
- `turnstile.go` — background Cloudflare Turnstile auto-clicker, started by `Connect` when `opts.Turnstile` is set (lives in `pkg/browser` to avoid an import cycle; `pkg/turnstile` is a thin alias kept for API compatibility).
- `process_unix.go` / `process_windows.go` — platform-specific process-group kill and graceful-shutdown signalling for `ChromeProcess`.
- Network request interception via the `Page.OnRequest` / `InterceptedRequest.{Continue,Respond,Abort}` API; `network_fingerprint_proxy.go`.
- Extensions: `extension_manager.go`, `extension_installer.go`, `advanced_extension_injector.go`.
- Xvfb: `xvfb.go` (real, Linux) vs `xvfb_stub.go` (build-tag stub for non-Linux).

**Auxiliary `cmd/` tools** (each its own `package main`): `monitor` (resource/leak monitoring via `BrowserPool`), `stability_test` (long-running concurrency soak), `fingerprint_collector` (scrape real browser fingerprints into JSON), `fingerprint_stats` (config-pool combinatorics report).

## Conventions specific to this repo

- Code comments and the `dom/*.md` design docs are largely in Chinese; match that style when editing existing files.
- Because there are two page implementations, **a change to page behavior usually needs to be made in both `connector.go` (CDPPage) and `cdp_custom.go` (CustomCDPPage)** — they intentionally mirror each other. Grep both before assuming one covers it.
- `ConnectOptions` is the single wide config struct (`types.go`); new knobs are added there and threaded into launcher/connector, not via new constructors.
