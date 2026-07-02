# Sludge Exploder

**The internet is good. Recommendation algorithms are bad.**

A brute, barbaric, unsophisticated feed blocker for Chrome and Firefox.

The knowledge economy is worth protecting. Wikipedia, documentation, GitHub, direct search results, articles you chose to read — these are worth having. What isn't worth having is the feed: algorithmically-selected content engineered to maximize time-on-site rather than inform you of something. This extension hides feed elements while leaving intentional navigation intact.

Because I'm an addict. None of the available extensions were configurable enough.

## Config Reference

`config.js` assigns an array of site rules to `globalThis.SLUDGE_CONFIG`. The included config is an opinionated starting point — modify it freely. The friction is the point.

```js
globalThis.SLUDGE_CONFIG = [
  {
    "domain": "example.com",
    "selectors": [".feed", ".recommendations"],
    "permablock_selectors": [".always-hidden"],
    "allowWindow": { "start": 15, "end": 23 },
    "paths": [
      {
        "path": "/allowed-section/",
        "selectors": [],
        "allowWindow": null
      }
    ]
  }
];
```

| Field | Description |
|-------|-------------|
| `domain` | Domain to match. Subdomain-aware and specificity-based: `reddit.com` also matches `old.reddit.com`, but an explicit `old.reddit.com` rule wins. |
| `selectors` | CSS selectors to hide when outside the `allowWindow`. Use `["body"]` to block the whole page. |
| `permablock_selectors` | CSS selectors hidden regardless of `allowWindow` — for content never worth seeing (e.g. Shorts, autoplay recommendations). |
| `allowWindow` | `{ "start": H, "end": H }` in 24h local time. When the current hour is in range, `selectors` are suppressed. Overnight windows like `{ "start": 22, "end": 2 }` are supported. `null` means always block. |
| `paths` | Path-specific rule overrides. The most specific matching path wins, then merges with the domain config; `permablock_selectors` from both levels are combined. Use path rules to allow specific sections of an otherwise-blocked site. |

**Path matching:** exact match, or prefix match at a proper segment boundary. Patterns ending in `/` match the slashless path and all subpaths (e.g. `/jobs/` matches `/jobs` and `/jobs/software-engineer-123`). Patterns not ending in `/` won't partially match segments (e.g. `/lit` won't match `/literature`).

Validate config before reloading:

```sh
node validate-config.js
```
