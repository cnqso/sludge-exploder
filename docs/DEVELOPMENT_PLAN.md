# Sludge Exploder — Staged Development Plan

> Companion to [`ENFORCEMENT.md`](./ENFORCEMENT.md). This is written for someone
> picking up the work cold (think: a capable intern). Every stage has a clear
> **goal**, concrete **tasks**, an unambiguous **Definition of Done** you can
> check, and a **demo script** to prove it. Build **top-down**: get the
> extension solid, then extension↔UI, then extension↔UI↔daemon, then harden.
>
> Golden rule: **a stage is not done until its demo script works on a clean
> machine.** "It works on my setup" does not count.

---

## Stage map

| Stage | Slice working end-to-end | Lockdown power |
|---|---|---|
| 0 | Repo & toolchain scaffolding | none |
| 1 | **Extension** standalone, config-driven, status-reporting | none |
| 2 | **Extension ↔ UI** (preferences, setup helper, dummy Lock) | none |
| 3 | **Extension ↔ UI ↔ Daemon** (heartbeat, then kill, real Lock) | soft |
| 4 | Friction & legitimacy (force-install policy, signed, removable) | intermediate |
| 5 | Force-install & distribution (Web Store, Safari, Windows) | shippable |

Stages 1–3 are the bulk of the product value and carry **zero lockdown risk** —
nothing can trap the user yet, so they're safe to iterate fast. Stages 4–5 add
*friction* (intermediate-user-grade, signed, always-removable) — not hard
lockdown. We deliberately never build self-defense; see `ENFORCEMENT.md` §6.

---

## Stage 0 — Foundations

**Goal:** one repo, one toolchain, everything builds and runs empty.

**Tasks**
- Create a Go module (`go.mod`) for the native side. Lay out:
  ```
  /extension        (existing zapper.js, config.js, manifest.json)
  /app              (Go + webview UI)
  /daemon           (Go privileged enforcer)
  /shared           (Go: IPC types, config schema, protocol constants)
  /docs             (these files)
  ```
- Add a top-level `Makefile`/`Taskfile` with `build`, `run-app`, `run-daemon`.
- Get a hello-world webview window opening from `/app` via
  [`webview/webview_go`].

**Definition of Done**
- [ ] `make build` produces an app binary and a daemon binary with no errors.
- [ ] `make run-app` opens an empty native window titled "Sludge Exploder."
- [ ] `/shared` compiles and is imported by both `/app` and `/daemon`.

**Demo:** run `make run-app` on a clean checkout → a window appears.

**Out of scope:** any real UI, any IPC, any blocking changes.

---

## Stage 1 — Extension, standalone and observable

**Goal:** the existing blocker keeps working, but is now **config-driven from
storage** and can **report its status** — the two hooks the rest of the system
needs. No native code involved yet.

**Why first:** the extension is the product's reason to exist and the only piece
that already works. Lock its contract down before anything depends on it.

**Tasks**
- Refactor `zapper.js` to read rules from `chrome.storage.local` (key e.g.
  `SLUDGE_CONFIG`), falling back to the bundled `config.js` on first run. Keep
  all existing matching logic (domain, paths, allowWindow, permablock).
- Add a `getStatus()` message handler in the extension returning
  `{ extId, version, configHash, rulesActive: <count> }`.
- Keep `validate-config.js` working against the new storage-shaped config.

**Definition of Done**
- [ ] With no stored config, the extension behaves exactly as today (bundled
      fallback). Verified on a site currently in the config.
- [ ] Writing a new config to `chrome.storage.local` and reloading the tab
      changes what's blocked — *without* editing any file.
- [ ] Sending `getStatus` from the extension console returns correct values.
- [ ] `node validate-config.js` passes on the bundled config.

**Demo:** load the unpacked extension → confirm a known site is blocked → in
the service-worker console, write a modified config to storage → reload the
target tab → the change is visible.

**Out of scope:** UI, native messaging, heartbeat (added in Stage 3).

---

## Stage 2 — Extension ↔ UI

**Goal:** a real preferences app. The user can pick **what to block**, **how
long**, get **per-browser setup help**, and press a **Lock button** — which for
now just shows a confirmation. The UI writes config the extension obeys.

**Why now:** this is where the product becomes usable by a human who can't read
JSON. The whole preference flow the user described lives here. It's also still
lockdown-free, so we can iterate UX freely.

**Tasks**
- **Config bridge.** Stand up the Native Messaging host (thin Go binary) and a
  `SetConfig` path so the app can push rules into the extension's storage. (This
  is the first native↔extension link; the daemon comes in Stage 3. For now the
  *app* can host the native-messaging endpoint directly.)
- **Preference UI** (HTML/JS in the webview), in the order the user flows
  through:
  1. **Which apps** — pick from a starter catalog (YouTube, Reddit, X, …) plus
     "add custom domain."
  2. **Which parts** — per app, toggle the named element groups (e.g. YouTube:
     "home feed," "comments," "Shorts") mapped to selector sets.
  3. **How long** — duration picker for the eventual lock.
  4. **Setup helper** — detect installed browsers; for each, show steps to
     install the extension and grant permissions, with a live "extension
     detected ✓/✗" check.
  5. **Lock button** — prominent; on click shows a summary + confirm dialog.
     **Does not actually lock yet** (no daemon) — it just records intent.
- **Status panel** — show the extension's `getStatus` live in the UI.

**Definition of Done**
- [ ] Selecting apps/parts in the UI and saving changes what the extension
      blocks on a live tab, no file editing, no extension reload.
- [ ] The setup helper correctly shows ✓ when the extension is installed in a
      browser and ✗ when it isn't.
- [ ] The duration picker persists a chosen lock length.
- [ ] The Lock button shows an accurate summary ("Blocking 4 apps for 2h") and a
      confirm dialog; confirming logs intent but nothing is enforced.
- [ ] Killing and reopening the app preserves the saved preferences.

**Demo:** open the app → pick YouTube, disable "home feed" only → save → open
YouTube, feed is gone but search/video work → return to app, see status ✓ → set
2h → press Lock → confirmation appears (and nothing is actually locked).

**Out of scope:** any enforcement, the daemon, heartbeat, resisting removal.

---

## Stage 3 — Extension ↔ UI ↔ Daemon (soft enforcement)

**Goal:** introduce the privileged-shaped daemon and wire the **heartbeat** and
the **real Lock**. Enforcement is *soft*: the daemon can close browsers and the
lock has a real timer, but it does not yet resist its own removal — so it's
still safe to test. This is the first time the system has teeth, intentionally
dull ones.

**Why now:** with extension and UI proven, we add the one component that makes
locking real, and we de-risk the hardest integration (the heartbeat) before
investing in OS-level hardening.

**Tasks**
- **Daemon skeleton** (`/daemon`): the shared core — lock state machine,
  persisted lock state, IPC server (unix socket / named pipe), heartbeat server.
  Define the `Enforcer` interface; ship a `darwin` backend that, at first, only
  **logs** "would kill browser X."
- **Move the native-messaging host to relay to the daemon** instead of the app,
  so the heartbeat reaches the privileged process (per `ENFORCEMENT.md` §5.1).
- **Heartbeat client in the extension** (the addition deferred from Stage 1):
  every 5s send `{alive, extId, version, configHash}`; receive `{lockState,
  until}` back.
- **App → daemon control:** `StartLock{duration}` and `GetStatus` over IPC. The
  Lock button now actually starts a timed lock; the UI shows a live countdown
  read from the daemon.
- **Flip the darwin backend from log to act:** kill a hard-coded "uncontrollable"
  browser (use a throwaway browser as the test target, never the dev's main
  one), and close a controllable browser when its heartbeat goes `MISSING`.

**Definition of Done**
- [ ] Daemon running → extension heartbeat visible in daemon logs within 5s of a
      browser opening.
- [ ] Disable the extension → within N intervals the daemon logs `MISSING` and
      (in act mode) closes that browser.
- [ ] Pressing Lock with a 2-minute duration starts a countdown the UI reflects;
      after 2 minutes the daemon returns to `UNLOCKED` on its own.
- [ ] Lock state survives killing the app *and* rebooting: after reboot, if still
      within the window, the daemon reports `LOCKED` with the correct remaining
      time.
- [ ] Launching the designated "uncontrollable" test browser → daemon kills it.
- [ ] `StopLock` is refused by the daemon while `LOCKED`.

**Demo:** start daemon → open browser, see heartbeat → press Lock for 2 min →
try to disable the extension, browser closes → try `StopLock`, refused →
wait out the timer (or reboot mid-window and confirm it resumes) → lock clears.

**Out of scope:** resisting removal of the daemon itself, force-install,
publishing. The daemon can still be quit/uninstalled by hand here — that's fine.

---

## Stage 4 — Friction & legitimacy (intermediate-user-grade, not malware)

**Goal:** make a lock *meaningfully inconvenient to escape for a casual or
intermediate user*, while keeping the app obviously-legitimate (signed, honest,
removable). This is **friction, not lockdown** — read `ENFORCEMENT.md` §6 before
starting. We are deliberately *not* building self-defense.

**Why now:** only after the full loop is proven do we add the force-install
policy and the start-on-login lifecycle. We do it carefully because every
enforcement behavior here is also a way to look like malware if overdone.

**Tasks**
- **Run the daemon as a normal start-on-login service:** `LaunchAgent`/Daemon
  (macOS) / Windows Service installed via a signed helper (`SMAppService` on
  macOS). It starts on login; it does **not** re-spawn itself when killed.
- **Write & re-assert the force-install policy** from the daemon: `.mobileconfig`
  / registry forcelist for the controllable browsers. Re-assert on a **relaxed
  cadence** (e.g. every few minutes) if removed — enough friction to outlast an
  impulse, not a keystroke-by-keystroke fight.
- **Refuse the in-app `StopLock` while LOCKED** (that's the product), but **do
  not block the uninstaller** — uninstalling is always allowed; it's just
  multi-step friction, not a wall.
- **Browser allowlist** of controllable browsers. For an *uncontrollable*
  browser, prefer a **visible "not supported during a block" warning**; reserve
  hard process-closing for the few common cases and always with a clear
  in-app explanation.
- **Signing & notarization first-class:** the build must pass Gatekeeper with no
  warnings. Treat any "this feels like malware" moment in testing as a bug.

**Definition of Done**
- [ ] A force-installed extension shows "Installed by your organization" with no
      disable/remove control in a controllable browser.
- [ ] Deleting the config profile / registry key while locked → daemon
      re-asserts within its relaxed cadence; the disable toggle returns then
      disappears again. (We accept the brief window — we're not fighting frame by
      frame.)
- [ ] `StopLock` is refused in-app while locked; the **uninstaller still runs**
      and cleanly removes everything at any time.
- [ ] Killing the daemon stops enforcement (expected — we don't re-spawn); it
      comes back on next login.
- [ ] The signed build launches with no Gatekeeper/SmartScreen warning.
- [ ] An intermediate-user escape (open another installed browser; click around
      System Settings for the toggle) is meaningfully inconvenient, and every
      enforcement action the user sees has a plain-language explanation.

**Demo (disposable VM/test machine, never a daily driver):** lock for 5 min →
try the casual/intermediate escapes from `ENFORCEMENT.md` §6 → each is
inconvenient and clearly explained → confirm the uninstaller *does* work even
while locked (friction, not a trap) → wait out the timer.

**Out of scope:** any self-defense/anti-removal arms race (explicitly rejected),
Web Store publishing, Safari, Windows parity polish (next stage).

---

## Stage 5 — Force-install & distribution

**Goal:** ship something a stranger can install. Close the publishing dependency
that force-install requires, and reach platform parity.

**Tasks**
- **Publish the extension** to the Chrome Web Store (and Firefox AMO); set up the
  update URL the forcelist points at. Keep a self-hosted update manifest as
  backup.
- **Windows parity:** Windows Service + registry `ExtensionInstallForcelist`,
  named-pipe IPC, `taskkill` backend — the second half of the build-tagged
  daemon.
- **macOS code signing & notarization**; signed installer; signed privileged
  helper.
- **Safari track (separate):** Full Disk Access flow to verify/maintain the
  Safari extension (no forcelist exists).

**Definition of Done**
- [ ] A clean macOS machine can install from a signed package, force-install the
      published extension, and complete a full locked session end-to-end.
- [ ] The same is true on a clean Windows machine.
- [ ] Gatekeeper/notarization passes with no warnings on macOS.
- [ ] Uninstall is clean on both platforms when unlocked.

**Demo:** hand the installer to someone who has never seen the repo → they
install, pick blocks, lock for an hour → every escape is defended → at the hour
it releases and uninstalls cleanly.

**Out of scope:** Linux (post-MVP), MDM/enterprise supervision, mobile.

---

## Sequencing & risk notes for whoever builds this

- **Do not skip the demo scripts.** Each is the acceptance test; on a clean
  machine they catch the "works on my box" failures that matter most for an
  enforcement product.
- **Stages 1–3 are safe; 4–5 touch system policy.** Uninstall always works (no
  trap by design), but from Stage 4 on you're installing config profiles /
  registry policy and closing browsers — test *only* on a disposable VM or a
  machine you can wipe until those paths are proven clean.
- **The Web Store dependency (Stage 5) has external lead time** — start the
  listing/review process early, in parallel with Stage 4.
- **Validate `webview_go` in Stage 0–2.** If it fights back, fall back to the
  localhost-served UI rather than reaching for Electron/Wails.

[`webview/webview_go`]: https://github.com/webview/webview_go
