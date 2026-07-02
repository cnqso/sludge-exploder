# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

**Sludge Exploder** is a browser extension (Manifest V3, compatible with Chrome/Edge/Firefox) that blocks or hides content on specific websites using CSS injection. It runs as a content script on all URLs and reads its configuration from `config.js`.

There is no build step, no package manager, no test suite. The extension is loaded directly as an unpacked extension in the browser.

## Reloading After Changes

After modifying any file, reload the extension to apply changes:
- **Chrome/Edge**: Go to `chrome://extensions/` → click the refresh icon next to the extension
- **Firefox**: Go to `about:debugging` → click "Reload"

Run `node validate-config.js` before reloading after config changes.

## Architecture

The entire logic lives in two files:

- **`config.js`** — Loaded before `zapper.js`; assigns the site rules to `globalThis.SLUDGE_CONFIG`.
- **`zapper.js`** — Content script injected into every page at `document_start`. It finds the most specific matching domain entry, resolves the most specific path override, checks the `allowWindow` time range, then injects a `<style>` tag to hide matched elements via CSS.
- **`validate-config.js`** — Node-based validation for the config shape and manifest script ordering.

Each config entry can have:
  - `domain` — matched against `window.location.hostname` (subdomain-aware)
  - `selectors` — CSS selectors to hide when the site is not in the allowed time window
  - `permablock_selectors` — CSS selectors hidden regardless of time window
  - `allowWindow` — `{ start: hour, end: hour }` in 24h local time; if current hour is in range, `selectors` are not applied (only `permablock_selectors` still apply)
  - `paths` — array of path-specific overrides; each path entry merges with the domain entry (path-level `permablock_selectors` are merged additively with domain-level ones)

## config.js Schema

```json
[
  {
    "domain": "example.com",
    "selectors": ["body"],
    "permablock_selectors": [".some-element"],
    "allowWindow": { "start": 15, "end": 23 },
    "paths": [
      {
        "path": "/some-path/",
        "selectors": [],
        "allowWindow": { "start": 15, "end": 23 },
        "permablock_selectors": [".path-specific"]
      }
    ]
  }
]
```

Path matching: exact match OR prefix match with proper segment boundary (no partial segment matches). Patterns ending in `/` match the slashless path and all subpaths. Overnight allow windows are supported.
