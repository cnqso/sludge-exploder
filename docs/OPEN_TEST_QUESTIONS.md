# Open Test Questions

Things that are implemented and pass automated/protocol-level verification,
but need a manual pass by a human in front of the actual UI/OS before
they're considered fully confirmed. Move an item out of this file (delete
the entry) once it's been checked off; add a note here if it turns out
broken instead.

## Stage 3

- **Self-focus on "at risk," macOS (`app/focus_darwin.go`).** The
  daemon-side risk tracking and `osascript` command are both verified
  independently (protocol-level: `risk` field populates correctly in
  `GetStatus` responses with a live countdown; `osascript` itself confirmed
  to bring a real process to the front). Not yet verified: whether the app
  window actually visibly pops to the front, and the red at-risk banner
  renders correctly with a live countdown, during a real end-to-end
  scenario (extension enabled + connected in Chrome, then disabled, while
  locked). Needs a human at the keyboard to watch the window behavior —
  this can't be driven by automation in this environment.

## Stage 4 (Windows)

**None of this has ever run.** Everything below compiles for
`GOOS=windows` (verified here via cross-compilation for `daemon`/`shared`/
`app/nmhost`, and via a stubbed-`main.go` type-check for `app` itself, since
`webview_go`'s WebView2 backend needs headers the local mingw-w64 toolchain
doesn't have) but has zero runtime verification. Treat all of it as
unproven until checked on the actual Windows device.

- **IPC over named pipes** (`shared/ipc_windows.go`, via `go-winio`).
  Heartbeat and control channels both need to actually connect, send, and
  receive correctly — the darwin equivalent was verified live with real
  socket traffic; the Windows named-pipe path never has been.
- **Browser detection** (`shared/browsers_windows.go`). Install paths for
  Chrome/Edge/Brave/Firefox are my best understanding of default Windows
  install locations, not verified against a real install. Chrome
  especially may only be found in one of the three candidate paths
  depending on how it was installed (per-user vs. per-machine).
- **Native-messaging registration** (`app/hostinstall_windows.go`). Writes
  a manifest file + `HKCU` registry value; never confirmed that Chrome/Edge
  on Windows actually discovers and connects through it.
- **`tasklist`/`taskkill` Enforcer** (`daemon/enforcer_windows.go`). CSV
  parsing for `IsRunning` in particular is exactly the kind of thing that
  can silently break on locale/Windows-version differences in `tasklist`'s
  output format.
- **`nmhost`'s Windows browser detection** (parent-PID lookup via
  `tasklist`). Same CSV-parsing caveat as above.
- **Self-focus on "at risk," Windows (`app/focus_windows.go`).** Real
  implementation via `user32.dll`'s `SetForegroundWindow` (pure Go, no cgo,
  using the HWND `webview_go`'s `Window()` exposes) -- but it has never
  run. Confirm the window actually comes to the front, not just that the
  syscall doesn't crash.
- **Start-on-login** (`app/startup_windows.go`, `HKCU\...\Run`). Never
  confirmed the daemon actually launches at login, or that the registry
  value is trivially removable the way it's supposed to be.
- **Force-install policy registry write** (`daemon/forcelist_windows.go`).
  The registry write itself is unverified, and — separately, expected —
  it will have **no observable effect** on Chrome's actual behavior until
  the extension is published (Stage 5). The Brave policy key
  (`WindowsForcelistKey` in `shared/browsers_windows.go`) is a best-effort
  guess I could not verify against Brave's real enterprise policy docs from
  this environment.
