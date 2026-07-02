# Sludge Exploder — Enforcement Architecture

> Design doc for adding Cold-Turkey-grade lockdown to Sludge Exploder while
> keeping its surgical, element-level blocking. This document explains the
> **why** behind every component, not just the **how**. The companion
> [`DEVELOPMENT_PLAN.md`](./DEVELOPMENT_PLAN.md) turns it into staged work.

---

## 1. What we have and what we're adding

Today Sludge Exploder is a pure Manifest V3 content script. `config.js` assigns
site rules to `globalThis.SLUDGE_CONFIG`; `zapper.js` injects a `<style>` tag
that hides matched elements. There is no native code, no build step, nothing
running outside the browser.

That design is also its ceiling. **A browser extension can never lock itself
down.** Anything the extension does, the user can undo from
`chrome://extensions` (toggle it off), from the profile menu, or by launching a
different browser. For a focus/self-control product that is fatal: the moment
of weakness is exactly the moment the user reaches for the off switch.

The product thesis is: **surgical element-blocking (our differentiator) +
Cold-Turkey-grade lockdown (the moat).** Freedom and Cold Turkey have the
lockdown but block whole sites bluntly. Nobody pairs "hide the feed but keep the
video" with "you cannot turn this off until the timer ends." That combination is
the wedge.

To get there we add a **privileged native enforcer** that lives outside the
browser and treats the extension as something it *protects and verifies*, not
something it trusts.

---

## 2. Why an extension alone cannot enforce (the core constraint)

Three independent escape hatches exist that no extension can close from inside:

1. **Disable/remove the extension.** `chrome://extensions` → toggle. The
   extension's own code stops running the instant you do this, so it cannot
   "defend" against it.
2. **Switch browsers.** Block Chrome all you like; the user opens Firefox, Orion,
   a freshly-downloaded Chromium, or an Electron app with a webview.
3. **Use a private window / different profile** where the extension isn't loaded.

The only actor that can close these hatches is a process running **with OS
privilege, outside any browser**, that can (a) make the browser itself refuse to
disable the extension, and (b) observe and act on the whole system — including
killing browsers it cannot control. That process is the **enforcer daemon**.

This is precisely the split we observed in Cold Turkey: the *extension* does the
blocking, and a *desktop app* closes your browser if the extension isn't active.
The desktop app is the enforcer; the extension is the worker.

---

## 3. The three-tier architecture

```
┌─ TIER 1: Extension (Chrome / Firefox) ─────────────┐
│  • Surgical CSS element-hiding (existing zapper.js) │  ← our differentiator
│  • Heartbeat client: proves "I am alive & enabled"  │
└───────────────┬─────────────────────────────────────┘
   Native Messaging (browser-spawned, origin-locked)
┌───────────────┴─ TIER 2: User App (Go + webview) ──┐
│  • Local HTML UI in a system webview (no Electron)   │  ← config + control
│  • Preference flow: what to block / how long         │
│  • Per-browser setup helper                          │
│  • The "Lock" button                                 │
└───────────────┬─────────────────────────────────────┘
   Local IPC (unix socket / named pipe, authenticated)
┌───────────────┴─ TIER 3: Enforcer Daemon (Go) ─────┐
│  • PRIVILEGED: root LaunchDaemon / Windows Service   │  ← the keystone
│  • Holds lock state & timer (survives app/UI death)  │
│  • Verifies extension heartbeat + policy presence    │
│  • Kills browsers it cannot control                  │
│  • Writes & re-asserts force-install policy (relaxed)│
│  • Starts on login; persists the lock across reboot  │
└──────────────────────────────────────────────────────┘
```

**Why three tiers and not two?** Privilege separation. The UI is user-facing,
frequently updated, and must *not* run as root (a webview rendering HTML as root
is a liability). The daemon is small, rarely changed, and is the only thing that
needs privilege. Keeping them separate means the attack surface that runs as
root is tiny and auditable, and the daemon keeps enforcing even if the UI app is
force-quit — which a motivated user *will* try.

**Why is "policy" not its own tier?** The macOS configuration profile and the
Windows registry forcelist are not components you maintain by hand — they are
**state the daemon writes and continuously re-asserts**. If the user deletes the
profile, the daemon rewrites it. So policy is data owned by Tier 3, not a fourth
moving part.

---

## 4. Component deep-dives

### 4.1 Tier 1 — The extension (mostly built)

Unchanged in spirit: it still reads rules and injects CSS. Two additions:

- **Heartbeat client.** On a timer (e.g. every 5s) the extension opens a Native
  Messaging connection to the daemon's host and sends `{alive, extId, version,
  configHash}`. If the extension is disabled or removed, the connection dies and
  the daemon notices within one interval.
- **Config sync.** Config stops living only in a bundled `config.js`. The app
  (Tier 2) becomes the source of truth and pushes rules into the extension's
  `chrome.storage` via Native Messaging. `zapper.js` reads from storage with the
  bundled config as a first-run fallback. *Why:* the UI must be able to edit "what
  parts of which apps" and have the extension obey without a reload.

**Why Native Messaging for the heartbeat instead of `fetch('localhost')`?**
Security, and it's the crux of the whole threat model. A localhost socket can be
hit by *any* process, so a user could run a 10-line script that fakes the
heartbeat while the extension is actually disabled — defeating enforcement.
Native Messaging hosts are registered with `allowed_origins` pinned to the
extension ID, and **the browser itself spawns the host process**. A non-browser
process cannot impersonate that channel. So Native Messaging gives us a much
stronger signal that *the real extension, in a real browser, is really running*.

### 4.2 Tier 2 — The user app (Go + system webview)

A normal user-privilege desktop app. Responsibilities:

- Render the **preference UI** (HTML/CSS/JS) in the OS's built-in webview
  (`WKWebView` on macOS, `WebView2` on Windows) via a thin Go binding such as
  [`webview/webview_go`]. The Go process serves the UI and exposes a few bound
  functions / a tiny local HTTP API the page calls.
- Walk the user through: **which apps** to block → **which parts** of each app →
  **how long** the lock lasts → **per-browser setup** (install the extension,
  grant permissions) → the **Lock** button.
- Talk to the daemon over local IPC to start/stop locks and read status.

**Why a webview and explicitly not Wails or Electron.** Electron ships a ~150MB
Chromium per app — ironic for a tool that polices browsers, and a second runtime
on top of our Go daemon. Wails is lighter but, in our experience, becomes a
build/tooling quagmire. A raw system-webview binding keeps the UI as plain,
portable HTML served by the *same Go process family* as the daemon — one
language end to end, ~10–20MB, the OS provides the renderer. If even the webview
binding proves fussy, the fallback is "serve the UI on localhost and open it,"
but we avoid that while a browser-policing product is the whole point.

**Why the app is separate from the daemon.** See privilege separation in §3. The
app can crash, update, or be quit; the lock lives in the daemon and is
unaffected.

### 4.3 Tier 3 — The enforcer daemon (Go, privileged)

The keystone. One Go codebase, cross-compiled, with a small platform seam:

```
enforcer/
  main.go              // shared: lock state machine, heartbeat server, scheduler
  enforcer.go          // interface: KillBrowser / WritePolicy / ResistRemoval / ...
  enforcer_darwin.go   // LaunchDaemon, .mobileconfig profile, pkill, Full Disk Access
  enforcer_windows.go  // Windows Service, registry ExtensionInstallForcelist, taskkill
  enforcer_linux.go    // later
```

Responsibilities:

- **Own the lock.** A state machine (`UNLOCKED → LOCKED(until=T) → UNLOCKED`)
  persisted to privileged storage so it survives reboots and app death.
- **Verify guarantees on a tick** (~every few seconds): Is the extension
  heartbeating? Is the force-install policy present? Are only controllable
  browsers running?
- **Act on violations.** Uncontrollable browser running (e.g. raw Chromium) →
  kill it. Policy profile deleted → rewrite it. Extension heartbeat missing in a
  controllable browser → close that browser.
- **Apply light friction while LOCKED.** Re-assert the policy on a relaxed
  cadence if it's removed; refuse the in-app `StopLock`. Note what's *not* here:
  no re-spawn-on-kill, no uninstaller blocking — see §6 for why.

**Why one daemon and not two.** ~80% of the daemon is OS-agnostic (state
machine, timers, heartbeat server, the *decision* logic). Only the *mechanisms*
differ per OS, and Go's build tags isolate them behind one interface. It's "one
daemon, two backends," not a fork — the genuine cross-platform cost is small and
contained.

**Why Go.** Single static binary per platform, trivial cross-compilation,
first-class Windows Service and macOS daemon stories, and it's the same language
as the app so the IPC/protocol/config types are shared code.

### 4.4 Policy state — force-install (data owned by Tier 3)

The mechanism that makes the *browser itself* refuse to disable the extension:

- **macOS:** an Apple Configuration Profile (`.mobileconfig`) setting Chrome's
  `ExtensionInstallForcelist` (payload domain `com.google.Chrome`, materialized
  under `/Library/Managed Preferences/`). Edge/Brave/Firefox have equivalent
  policy keys. A force-installed extension shows "Installed by your
  organization" with no disable/remove control.
- **Windows:** the registry key
  `HKLM\SOFTWARE\Policies\Google\Chrome\ExtensionInstallForcelist` (and
  per-browser equivalents). Notably *easier* than the macOS profile.
- **Safari:** no forcelist. Cold Turkey leans on **Full Disk Access** to verify
  and maintain the Safari extension instead. Treat Safari as a later, separate
  track.

**Hard truth about force-install:** it pulls the extension from an *update URL* —
practically the Chrome Web Store (or a self-hosted update manifest). Our current
"load unpacked" flow **cannot** be force-installed. Shipping locked enforcement
therefore implies **publishing the extension**. This is a real dependency, not a
detail, and it gates the final stage.

---

## 5. The protocols

### 5.1 Extension ↔ Daemon — the heartbeat

- **Transport:** Native Messaging. The browser spawns a tiny native-messaging
  *host* (a thin Go binary) registered to our extension ID; the host relays to
  the long-running daemon over the local IPC socket.
- **Payload (ext → daemon):** `{ alive: true, extId, version, configHash, ts }`.
- **Reply (daemon → ext):** `{ lockState, until, configToApply? }` — lets the
  daemon push config and lock status back down the same channel.
- **Liveness rule:** daemon marks a browser's extension `ALIVE` on each beat;
  `MISSING` after N missed intervals → enforcement action for that browser.

### 5.2 UI App ↔ Daemon — control & status

- **Transport:** unix domain socket (macOS) / named pipe (Windows), with
  filesystem perms + a per-install shared secret so arbitrary processes can't
  drive the daemon.
- **Commands:** `StartLock{duration, profile}`, `GetStatus`, `SetConfig{rules}`,
  `RunSetupCheck{browser}`. Note there is intentionally **no `StopLock` while
  LOCKED** — that's the whole point.

### 5.3 Lock state machine

```
        StartLock(duration)
UNLOCKED ───────────────────────────▶ LOCKED(until = now + duration)
   ▲                                        │
   │   timer expires (until <= now)         │  StopLock → REFUSED while locked
   └────────────────────────────────────────┘
```

Persisted by the daemon to privileged storage. On boot the daemon reloads it; if
still within the window, it resumes enforcing immediately — closing the "just
reboot to escape" hole.

---

## 6. Design target — friction, not lockdown

This is **not a cybersecurity tool**, and the threat model is deliberately
modest. We are not defending against an adversary; we are defending the user
against their own ten-seconds-of-weakness. The goal is to **raise activation
energy past the impulse**, not to be unbreakable. Two reasons this is the right
call, not a cop-out:

1. **The user is the "attacker," and they're not a hacker.** The average target
   user can't download a release from a GitHub page, let alone boot into
   single-user mode or edit a LaunchDaemon plist. Friction that stops an
   *intermediate* user is friction that stops ~all of our users in the moment
   that matters.
2. **Hard lockdown looks like malware.** Root processes that re-spawn when
   killed, refuse to be uninstalled, and fight deletion are *literally the
   behavioral signature of malware* — they trip Gatekeeper/SmartScreen/AV, they
   alarm users, and they're a support and trust nightmare. The more
   "unbreakable" we make it, the more we look like the thing we're not. We
   explicitly will not go there.

So we calibrate to **tiers of user**, not to a determined attacker:

| User tier | Example escape attempt | Our stance |
|---|---|---|
| **Casual** | Clicks the extension's disable toggle | **Stop.** Force-install removes the toggle — the #1 impulse path, gone. |
| **Casual** | Quits the app, reboots | **Stop, cheaply.** Lock lives in the daemon; it persists across reboot. No self-defense heroics needed. |
| **Intermediate** | Opens a different browser they have installed | **Stop.** Force-install covers the common managed browsers; the daemon closes a controllable browser whose extension is missing. |
| **Intermediate** | Googles "how to remove a config profile," deletes it | **Slow down.** Daemon re-asserts the policy on a relaxed cadence (not a tight combat loop). Re-installing is more hassle than the impulse is worth. |
| **Determined / advanced** | Single-user mode, kills the daemon, edits plists, uses another machine or their phone | **Let them.** Out of scope by design. Cold Turkey concedes this too. |

**What we deliberately drop** (versus a hardline lockdown): no self-re-spawning
"you can't kill me" supervisor, no refusing-to-uninstall combat, no tight
re-assertion loop fighting the user keystroke-for-keystroke. These add the most
malware-like behavior for the least real-world benefit, because the users they'd
stop have already left the building. A normal LaunchAgent/Service that simply
*starts on login* is plenty.

**What friction we keep**, because it's high-leverage and legitimate-looking:
removing the disable toggle (managed/force-installed extension is a normal,
signed, above-board mechanism), closing a controllable browser when its
extension goes missing, persisting the lock across reboot, and a clear UI that
makes the *unlocked* uninstall path obvious and easy (the friction is during a
lock, not forever).

**Closing an uncontrollable browser** (e.g. a raw Chromium the policy can't
reach) is the one genuinely aggressive behavior we retain — but as a *narrow*
measure against the few common browsers, with a clear in-app explanation, not a
silent process-killing sweep. If it ever feels malware-adjacent in practice,
prefer a visible "this browser isn't supported during a block" warning over a
hard kill.

**Non-goals:** we are **not** Freedom. We deliberately do **not** block at the
network/proxy/VPN layer, because a proxy sees hosts, not DOM elements, and would
destroy our surgical-blocking differentiator. Browser-policy + light process
enforcement is the right layer for *element-level* blocking.

---

## 7. The cross-platform seam (what's shared vs per-OS)

| Concern | Shared (Go) | Per-OS backend |
|---|---|---|
| Lock state machine & persistence | ✅ | path/ACL of the store |
| Heartbeat server & liveness logic | ✅ | — |
| Kill *decision* (which browsers are uncontrolled) | ✅ | how to enumerate & kill processes |
| Force-install policy | format of rules | `.mobileconfig` vs registry |
| Daemon lifecycle | start-on-login logic | LaunchAgent/Daemon vs Windows Service |
| UI | 100% shared HTML/JS | webview binding picks `WKWebView`/`WebView2` |

The per-OS column is the only place a second implementation exists, and it's
small. Everything else is written once. Note "daemon lifecycle" here is plain
*start-on-login* supervision — **not** a re-spawn-on-kill self-defense loop,
which we deliberately don't build (see §6).

---

## 8. Key decisions, summarized

- **Friction, not lockdown.** Calibrate to stopping a casual/intermediate user in
  the moment of impulse — not to an adversary. Hard self-defense is dropped
  because it looks like malware and stops only users we've already lost (§6).
- **Three tiers, privilege-separated.** Only the small daemon runs as root.
- **Native Messaging for the heartbeat**, not localhost — it's the difference
  between a real liveness signal and a spoofable one.
- **One Go daemon with build-tagged OS backends**, not two daemons.
- **System webview, not Electron/Wails** — one language, ~10–20MB, OS-provided
  renderer.
- **Browser-policy + process enforcement, not a network proxy** — preserves
  surgical blocking.
- **Force-install requires publishing the extension** — a known, gating
  dependency for the final stage.

---

## 9. Open questions / risks to resolve as we build

1. **Web Store publishing & review.** Timeline and policy review for a
   force-installable listing; do we also self-host an update manifest as backup?
2. **macOS profile installation UX without an MDM.** Manually-installed profiles
   are user-removable in System Settings; our leverage is the daemon
   *re-asserting*, not the profile being unremovable. Confirm this holds on
   current macOS.
3. **`webview_go` maturity.** Validate it early (Stage 2); keep the
   "localhost-served UI" fallback in pocket.
4. **Code signing & notarization** (macOS) and a signed privileged-helper install
   (`SMAppService`) — real packaging work, scheduled before any public release.
5. **Firefox force-install** parity (`ExtensionSettings` policy via
   `policies.json` / profile) — confirm per-OS.
6. **Browser allowlist maintenance.** New browsers appear; the "controllable"
   list is an ongoing chore. Decide the update mechanism.
7. **Malware-perception / trust.** Even our *light* enforcement (closing a
   browser, installing a profile) can spook users and AV. Prioritize signing,
   notarization, and clear in-app explanations of every action; treat any
   behavior that "feels like malware" in testing as a bug, not a feature.

[`webview/webview_go`]: https://github.com/webview/webview_go
