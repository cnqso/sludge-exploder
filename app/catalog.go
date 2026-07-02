package main

// CatalogPart is one togglable named element group within an app (e.g.
// YouTube's "Home feed" or "Shorts"). Enabling it means "block this."
type CatalogPart struct {
	ID                  string   `json:"id"`
	Label               string   `json:"label"`
	Selectors           []string `json:"selectors,omitempty"`
	PermablockSelectors []string `json:"permablockSelectors,omitempty"`
}

// CatalogApp is one site in the starter catalog.
type CatalogApp struct {
	ID     string        `json:"id"`
	Label  string        `json:"label"`
	Domain string        `json:"domain"`
	Parts  []CatalogPart `json:"parts"`
}

// starterCatalog mirrors the rules already curated in extension/config.js,
// split into named, individually-togglable parts for the UI. Sites whose
// existing rules already block at the body level (Reddit's old UI, X,
// TikTok, Facebook, Instagram, ...) get a single "whole site" part. Path-
// scoped nuance from config.js (Instagram's per-section paths, Facebook
// Marketplace, 4chan /lit/) isn't modeled generically here yet -- that's a
// v2 catalog concern; for now those cases stay reachable via "custom
// domain," and the bundled config.js remains the fallback whenever storage
// is empty.
func starterCatalog() []CatalogApp {
	return []CatalogApp{
		{
			ID: "youtube", Label: "YouTube", Domain: "youtube.com",
			Parts: []CatalogPart{
				{ID: "home_feed", Label: "Home feed & recommendations", Selectors: []string{
					"ytd-rich-grid-renderer", "yt-lockup-view-model",
					"ytd-watch-next-secondary-results-renderer", "ytd-browse",
				}},
				{ID: "shorts", Label: "Shorts", PermablockSelectors: []string{
					"div[class='ytd-reel-video-renderer']", "ytd-reel-video-renderer", "#shorts-container",
				}},
			},
		},
		{
			ID: "reddit", Label: "Reddit", Domain: "reddit.com",
			Parts: []CatalogPart{
				{ID: "feed", Label: "Feed & sidebar", Selectors: []string{
					"shreddit-feed", "#feed-category-bar", "shreddit-gallery-carousel", "aside", "#left-sidebar",
				}},
			},
		},
		{
			ID: "old_reddit", Label: "Old Reddit", Domain: "old.reddit.com",
			Parts: []CatalogPart{{ID: "whole_site", Label: "Whole site", Selectors: []string{"body"}}},
		},
		{
			ID: "x", Label: "X / Twitter", Domain: "x.com",
			Parts: []CatalogPart{{ID: "whole_site", Label: "Whole site", Selectors: []string{"body"}}},
		},
		{
			ID: "tiktok", Label: "TikTok", Domain: "tiktok.com",
			Parts: []CatalogPart{{ID: "whole_site", Label: "Whole site", Selectors: []string{"body"}}},
		},
		{
			ID: "facebook", Label: "Facebook", Domain: "facebook.com",
			Parts: []CatalogPart{{ID: "whole_site", Label: "Whole site", Selectors: []string{"body"}}},
		},
		{
			ID: "instagram", Label: "Instagram", Domain: "instagram.com",
			Parts: []CatalogPart{{ID: "whole_site", Label: "Whole site", Selectors: []string{"body"}}},
		},
		{
			ID: "substack", Label: "Substack Notes", Domain: "substack.com",
			Parts: []CatalogPart{{ID: "notes_feed", Label: "Notes feed", PermablockSelectors: []string{
				`div[aria-label="Notes feed"]`,
			}}},
		},
	}
}

// buildConfig turns the user's catalog selections + custom domains into a
// SLUDGE_CONFIG-shaped rule list -- the same schema validate-config.js
// checks and zapper.js already understands.
func buildConfig(p Prefs, catalog []CatalogApp) []map[string]any {
	var rules []map[string]any

	for _, app := range catalog {
		enabledParts := p.Selections[app.ID]
		if len(enabledParts) == 0 {
			continue
		}

		var selectors []string
		var permablock []string
		for _, part := range app.Parts {
			if !enabledParts[part.ID] {
				continue
			}
			selectors = append(selectors, part.Selectors...)
			permablock = append(permablock, part.PermablockSelectors...)
		}
		if len(selectors) == 0 && len(permablock) == 0 {
			continue
		}

		rule := map[string]any{
			"domain":      app.Domain,
			"allowWindow": nil,
		}
		if len(selectors) > 0 {
			rule["selectors"] = selectors
		}
		if len(permablock) > 0 {
			rule["permablock_selectors"] = permablock
		}
		rules = append(rules, rule)
	}

	for _, cd := range p.CustomDomains {
		if cd.Domain == "" || len(cd.Selectors) == 0 {
			continue
		}
		rules = append(rules, map[string]any{
			"domain":      cd.Domain,
			"selectors":   cd.Selectors,
			"allowWindow": nil,
		})
	}

	return rules
}
