# Windows Handoff

You're picking up this project on a Windows machine after it was built
almost entirely on macOS. This doc is your briefing — read it before
touching anything. It exists because the agent that built the Windows
support has **zero Windows runtime access**: everything below compiled
cleanly (cross-compiled from macOS, or type-checked with a stubbed
`main.go` to route around a cgo dependency that can't cross-compile) but
**has never actually executed**. You are the first real test.

Read `CLAUDE.md`, `README.md`, and `docs/DEVELOPMENT_PLAN.md` first for
what this project is and how it's staged. This doc only covers what's
specific to picking it up fresh on Windows.

## Safety rules — read this before running anything with `--enforce`

This daemon can forcibly close a running browser process. That is the
entire point of the product, but it means testing it carelessly can close
a browser you or the machine's owner actually needed open. The macOS build
of this exact feature **actually closed the developer's real Firefox**
during automated testing, because a test script combined `SetEnforcement`
and an active lock without accounting for what that combination does in
the real world. Don't repeat that mistake:

1. **Never pass `--enforce` to the daemon without knowing exactly what
   browsers are currently running on the machine and being fine with any
   of them closing.** Check `tasklist` first.
2. **`--uncontrolled=<name>` must never be a real browser someone relies
   on.** If you need to test the "close an uncontrolled browser" path, use
   a disposable/throwaway process you spawn yourself, not Chrome/Edge/
   Firefox on this machine, unless you've confirmed with whoever owns this
   device that it's genuinely fine to close.
3. **Test log-only first.** Run the daemon with no `--enforce` flag and
   confirm the logs show the `[log-only] would close ...` lines you expect
   *before* ever adding `--enforce`.
4. **The app's Enforcement toggle is always available as a kill switch**
   (`app/api.go`'s `SetEnforcement`) — it works independent of lock state
   and stops all process-closing immediately, even mid-lock. If a test
   starts doing something you didn't expect, flip it off first and figure
   out why after.
5. There's also a 90-second startup grace period per browser session by
   default (`--grace-period`, see `daemon/enforce.go`) — a browser won't be
   closed the instant it looks "missing," only after that window elapses.
   Don't assume 0 grace when reasoning about timing.

## What's actually new here (Stage 4, Windows-first)

Full design rationale is in `docs/DEVELOPMENT_PLAN.md` and the git history,
but the concrete new pieces are:

- `shared/ipc.go` + `ipc_windows.go`: named pipes via `go-winio`, replacing
  Unix domain sockets. `HeartbeatSocketPath()`/`ControlSocketPath()`
  (`shared/paths_windows.go`) return pipe names like
  `\\.\pipe\SludgeExploderHeartbeat`, not filesystem paths.
- `shared/browsers_windows.go`: Chrome/Edge/Brave/Firefox install-path
  detection and native-messaging/forcelist registry key paths. **These
  paths are my best understanding of default Windows install locations,
  not verified against a real install** — if browser detection doesn't
  work, check `InstallPaths` here first.
- `app/hostinstall_windows.go`: writes the native-messaging manifest to a
  file under `%AppData%\SludgeExploder\hosts\` and points the browser at it
  via a `HKEY_CURRENT_USER` registry value (no admin needed).
- `daemon/enforcer_windows.go`: `tasklist`/`taskkill` backend.
  **CSV-output parsing is the most likely thing to break** on a Windows
  version/locale this wasn't tested against.
- `app/startup_windows.go`: start-on-login via
  `HKCU\...\Run\SludgeExploderDaemon`. Toggle lives in the app's Setup
  helper section.
- `daemon/forcelist_windows.go`: writes `ExtensionInstallForcelist`. This
  one is **expected to have no visible effect** right now — Chrome won't
  honor a forcelist entry for an unpublished, unpacked extension (see
  `docs/ENFORCEMENT.md` §4.4). Don't spend time debugging "why doesn't this
  remove the disable toggle" — it can't, yet, that's Stage 5's job
  (publishing to the Web Store).

## Building on Windows

```
go build ./...
```
or
```
make build
```
Either produces `bin\app.exe`, `bin\daemon.exe`, `bin\nmhost.exe` — the
Makefile was fixed to use `go env GOEXE` for the right suffix per platform
(a real bug caught before this handoff: `-o bin/app` with no extension
does *not* auto-append `.exe` on Windows, which would have silently broken
`nmhostPath()`/`daemonPath()`, both of which look for the `.exe`-suffixed
names specifically).

`./dev.sh` is bash — it won't run natively on Windows (Git Bash/WSL
could run it, but that's untested too). Until/unless someone builds a
`dev.ps1` equivalent, just run the two binaries manually in separate
terminals:
```
.\bin\daemon.exe
.\bin\app.exe
```

## What to actually test

Work through `docs/OPEN_TEST_QUESTIONS.md`'s "Stage 4 (Windows)" section
top to bottom — it's the authoritative list, written at the same time as
this handoff so it won't drift out of sync. Delete each item as you
confirm it, or replace it with a note if it's actually broken. Roughly, in
the order that makes sense to test:

1. Does `go build ./...` even succeed here? (First real signal — the
   cross-compilation checks done on macOS could easily have missed
   something a native Windows toolchain catches.)
2. Launch `daemon.exe`, confirm it creates the named pipes and doesn't
   crash. `tasklist | findstr daemon` to confirm it's running.
3. Launch `app.exe`, confirm the window opens and the Setup helper detects
   installed browsers correctly (checks `shared/browsers_windows.go`'s
   `InstallPaths`).
4. Load the unpacked extension in Chrome (`chrome://extensions` → Developer
   mode → Load unpacked → this repo's `extension/` folder). Confirm the
   app auto-registers the native-messaging host (check
   `HKCU\Software\Google\Chrome\NativeMessagingHosts\com.sludgeexploder.host`
   in `regedit`) and the extension actually connects (Setup helper: ✓, then
   `ALIVE`).
5. Repeat the full Stage 3 functional checklist on Windows: save
   preferences → live CSS reload on an open tab, lock → countdown →
   "Attempt early unlock" refused → wait it out → auto-unlocks, at-risk
   banner + self-focus during the grace window. Self-focus has a real
   Windows implementation (`app/focus_windows.go`, `user32.dll`'s
   `SetForegroundWindow` via `syscall.NewLazyDLL` — pure Go, no cgo, using
   the HWND `webview_go`'s `Window()` already exposes, captured once in
   `main.go`) but **it has never actually run** — confirm the window
   really does pop to the front, not just that it doesn't crash.
6. Only after all of the above works in log-only mode: test a real
   `--enforce` close, following the safety rules above.
7. Toggle start-on-login, reboot, confirm the daemon actually launches.
   Confirm it's cleanly removable (untoggle it, or delete the registry
   value/check Task Manager's Startup tab).

## Reporting back

Update `docs/OPEN_TEST_QUESTIONS.md` as you go. If something's broken,
leave enough detail to act on (exact error, `tasklist`/`regedit` state,
Windows version) rather than just "doesn't work" — whoever picks this back
up (agent or human) won't have the machine in front of them either.
