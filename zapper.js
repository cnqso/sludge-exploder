(async function () {
    const runtimeAPI = (typeof browser !== 'undefined' && browser.runtime)
        ? browser.runtime
        : (typeof chrome !== 'undefined' ? chrome.runtime : null);

    if (!runtimeAPI) {
        console.error('SLUDGE EXPLODER: Runtime API not found.');
        return;
    }

    const matchesDomain = (host, domain) => {
        if (!host || !domain) return false;
        const normalizedHost = host.toLowerCase();
        const normalizedDomain = domain.toLowerCase();
        return (
            normalizedHost === normalizedDomain ||
            normalizedHost.endsWith(`.${normalizedDomain}`)
        );
    };

    let CONFIG;
    try {
        const configUrl = runtimeAPI.getURL('config.json');
        const response = await fetch(configUrl);
        CONFIG = await response.json();
    } catch (error) {
        console.error('SLUDGE EXPLODER: Failed to load config.json', error);
        return;
    }


    const currentHost = window.location.hostname.toLowerCase();
    const currentPath = window.location.pathname;
    const now = new Date();
    const currentHour = now.getHours();

    const siteConfig = CONFIG.find(entry => matchesDomain(currentHost, entry.domain));

    if (!siteConfig) return;

    // Check if current path matches any path-specific rule
    // Path rules apply to the exact path and all subpaths
    let activeConfig = siteConfig;
    if (siteConfig.paths && Array.isArray(siteConfig.paths)) {
        const pathMatch = siteConfig.paths.find(pathRule => {
            const pathPattern = pathRule.path;
            // Match exact path
            if (currentPath === pathPattern) return true;
            
            // Match subpaths: if pattern ends with '/', match any path starting with it
            // e.g., "/lit/" matches "/lit/", "/lit/thread/123", "/lit/thread/123?page=2"
            if (pathPattern.endsWith('/') && currentPath.startsWith(pathPattern)) {
                return true;
            }
            
            // For patterns not ending with '/', ensure proper path segment boundary
            // e.g., "/lit" should match "/lit" and "/lit/" but not "/literature"
            if (currentPath.startsWith(pathPattern)) {
                const nextChar = currentPath[pathPattern.length];
                return nextChar === undefined || nextChar === '/' || nextChar === '?';
            }
            return false;
        });
        if (pathMatch) {
            // Use path-specific config, merging with domain-level permablock_selectors
            activeConfig = {
                ...siteConfig,
                ...pathMatch,
                // Merge permablock_selectors from both domain and path
                permablock_selectors: [
                    ...(siteConfig.permablock_selectors || []),
                    ...(pathMatch.permablock_selectors || [])
                ]
            };
        }
    }

    let isAllowed = false;
    if (activeConfig.allowWindow) {
        if (currentHour >= activeConfig.allowWindow.start && currentHour < activeConfig.allowWindow.end) {
            isAllowed = true;
        }
    }

    const permablockRules = (activeConfig.permablock_selectors || [])
        .map(selector => `${selector} { display: none !important; visibility: hidden !important; opacity: 0 !important; }`)
        .join(" ");

    const regularRules = (!isAllowed && activeConfig.selectors)
        ? activeConfig.selectors
            .map(selector => `${selector} { display: none !important; visibility: hidden !important; opacity: 0 !important; }`)
            .join(" ")
        : "";

    const cssRules = (permablockRules + " " + regularRules).trim();

    if (!cssRules) return;

    const style = document.createElement('style');
    style.id = 'zapper-block-style';
    style.textContent = cssRules;

    if (document.head) {
        document.head.appendChild(style);
    } else {
        document.documentElement.appendChild(style);
    }

    const blockedTarget = activeConfig !== siteConfig 
        ? `${siteConfig.domain}${currentPath}` 
        : siteConfig.domain;
    console.log(`SLUDGE EXPLODER: Blocked content on ${blockedTarget}`);

})();