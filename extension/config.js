globalThis.SLUDGE_CONFIG = [
  {
    "domain": "youtube.com",
    "permablock_selectors": [
      "div[class='ytd-reel-video-renderer']",
      "ytd-reel-video-renderer",
      "#shorts-container"
    ],
    "selectors": [
      "ytd-rich-grid-renderer",
      "yt-lockup-view-model",
      "ytd-watch-next-secondary-results-renderer",
      "ytd-browse"
    ],
    "allowWindow": null
  },
  {
    "domain": "old.reddit.com",
    "selectors": [
      "body"
    ],
    "allowWindow": null
  },
  {
    "domain": "reddit.com",
    "selectors": [
      "shreddit-feed",
      "#feed-category-bar",
      "shreddit-gallery-carousel",
      "aside",
      "#left-sidebar"
    ],
    "allowWindow": null
  },
  {
    "domain": "substack.com",
    "permablock_selectors": [
      "div[aria-label=\"Notes feed\"]"
    ],
    "selectors": [

    ],
    "allowWindow": {
      "start": 15,
      "end": 23
    }
  },
  {
    "domain": "facebook.com",
    "selectors": [
      "body"
    ],
    "allowWindow": null,
    "paths": [
      {
        "path": "/marketplace",
        "selectors": [],
        "allowWindow": null
      }
    ]
  },
  {
    "domain": "instagram.com",
    "selectors": [],
    "allowWindow": null,
    "paths": [
      { "path": "/",        "selectors": ["body"], "allowWindow": null },
      { "path": "/explore", "selectors": ["body"], "allowWindow": null },
      { "path": "/reels",   "selectors": ["body"], "allowWindow": null },
      { "path": "/direct",  "selectors": ["body"], "allowWindow": null },
      { "path": "/stories", "selectors": ["body"], "allowWindow": null }
    ]
  },
  { "domain": "tiktok.com",
    "selectors": [
      "body"
    ],
    "allowWindow": null
  },
  {
    "domain": "wsj.com",
    "selectors": [
      "body"
    ],
    "allowWindow": {
        "start": 15,
        "end": 24
      }
  },
  {
    "domain": "nytimes.com",
    "selectors": [
      "body"
    ],
    "allowWindow": null
  },
  {"domain": "washingtonpost.com",
    "selectors": [
      "body"
    ],
    "allowWindow": null
  },
  { "domain": "ft.com",
    "selectors": [
      "body"
    ],
    "allowWindow": null
  },
  { "domain": "bluesky.app",
    "selectors": [
      "body"
    ],
    "allowWindow": null
  },
  {
    "domain": "x.com",
    "selectors": [
      "body"
    ],
    "allowWindow": null
  },
  {
    "domain": "twitter.com",
    "selectors": [
      "body"
    ],
    "allowWindow": null
  },
  {
    "domain": "4chan.org",
    "selectors": [
      "body"
    ],
    "allowWindow": null,
    "paths": [
      {
        "path": "/lit/",
        "selectors": [
          "body"
        ],
        "allowWindow": {
          "start": 15,
          "end": 23
        }
      }
    ]
  }
];
