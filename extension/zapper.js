(function () {
    const BUNDLED_CONFIG = globalThis.SLUDGE_CONFIG;
    const STORAGE_KEY = 'SLUDGE_CONFIG';

    // Patch the history API immediately, before config even loads. This
    // catches same-world navigation calls; the URL poller below covers
    // frameworks that bypass the patch. Doing this synchronously means we
    // don't miss an early SPA navigation while config loads asynchronously.
    let onNavigate = () => {};

    const _push = history.pushState.bind(history);
    history.pushState = function (...args) { _push(...args); onNavigate(); };

    const _replace = history.replaceState.bind(history);
    history.replaceState = function (...args) { _replace(...args); onNavigate(); };

    window.addEventListener('popstate', () => onNavigate());

    const matchesDomain = (host, domain) => {
        if (!host || !domain) return false;
        const normalizedHost = host.toLowerCase();
        const normalizedDomain = domain.toLowerCase();
        return (
            normalizedHost === normalizedDomain ||
            normalizedHost.endsWith(`.${normalizedDomain}`)
        );
    };

    const getUrlKey = () => (
        window.location.pathname +
        window.location.search +
        window.location.hash
    );

    const pathMatches = (currentPath, pathPattern) => {
        if (!pathPattern || !pathPattern.startsWith('/')) return false;
        if (currentPath === pathPattern) return true;

        if (pathPattern === '/') return false;

        if (pathPattern.endsWith('/')) {
            const withoutTrailingSlash = pathPattern.slice(0, -1);
            return (
                currentPath === withoutTrailingSlash ||
                currentPath.startsWith(pathPattern)
            );
        }

        if (!currentPath.startsWith(pathPattern)) return false;

        const nextChar = currentPath[pathPattern.length];
        return nextChar === undefined || nextChar === '/';
    };

    const selectMostSpecificDomain = (host, config) => {
        return config
            .filter(entry => matchesDomain(host, entry.domain))
            .sort((a, b) => b.domain.length - a.domain.length)[0];
    };

    const selectMostSpecificPath = (paths, currentPath) => {
        if (!Array.isArray(paths)) return null;
        return paths
            .filter(pathRule => pathMatches(currentPath, pathRule.path))
            .sort((a, b) => b.path.length - a.path.length)[0] || null;
    };

    const isWithinAllowWindow = (allowWindow, currentHour) => {
        if (!allowWindow) return false;

        const { start, end } = allowWindow;
        if (start === end) return false;

        if (start < end) {
            return currentHour >= start && currentHour < end;
        }

        return currentHour >= start || currentHour < end;
    };

    // Storage first, bundled config.js as the first-run fallback. A
    // chrome.storage.local read can fail if the extension context has been
    // invalidated (e.g. mid-reload) or storage isn't available, in which
    // case we fall back the same as if storage were simply empty.
    async function loadConfig() {
        try {
            const stored = await chrome.storage.local.get(STORAGE_KEY);
            const storedConfig = stored[STORAGE_KEY];
            if (Array.isArray(storedConfig) && storedConfig.length > 0) {
                return storedConfig;
            }
        } catch (err) {
            console.warn('SLUDGE EXPLODER: storage read failed, using bundled config.', err);
        }
        return BUNDLED_CONFIG;
    }

    (async function main() {
        const currentHost = window.location.hostname.toLowerCase();

        let CONFIG = await loadConfig();
        if (!Array.isArray(CONFIG)) {
            console.error('SLUDGE EXPLODER: SLUDGE_CONFIG not found.');
            return;
        }

        let configHash = sludgeHashConfig(CONFIG);
        let siteConfig = selectMostSpecificDomain(currentHost, CONFIG);

        chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
            if (message?.type === 'getStatus') {
                sendResponse({
                    extId: chrome.runtime.id,
                    version: chrome.runtime.getManifest().version,
                    configHash,
                    rulesActive: CONFIG.length,
                });
            }
        });

        function applyBlocking() {
            let style = document.getElementById('zapper-block-style');

            if (!siteConfig) {
                if (style) style.remove();
                return;
            }

            const currentPath = window.location.pathname;
            const currentHour = new Date().getHours();

            let activeConfig = siteConfig;
            const pathMatch = selectMostSpecificPath(siteConfig.paths, currentPath);

            if (pathMatch) {
                activeConfig = {
                    ...siteConfig,
                    ...pathMatch,
                    permablock_selectors: [
                        ...(siteConfig.permablock_selectors || []),
                        ...(pathMatch.permablock_selectors || [])
                    ]
                };
            }

            const isAllowed = isWithinAllowWindow(activeConfig.allowWindow, currentHour);

            const permablockRules = (activeConfig.permablock_selectors || [])
                .map(s => `${s} { display: none !important; visibility: hidden !important; opacity: 0 !important; }`)
                .join(' ');

            const regularRules = (!isAllowed && activeConfig.selectors)
                ? activeConfig.selectors
                    .map(s => `${s} { display: none !important; visibility: hidden !important; opacity: 0 !important; }`)
                    .join(' ')
                : '';

            const cssRules = (permablockRules + ' ' + regularRules).trim();

            if (cssRules) {
                if (!style) {
                    style = document.createElement('style');
                    style.id = 'zapper-block-style';
                    (document.head || document.documentElement).appendChild(style);
                }
                style.textContent = cssRules;
                const target = activeConfig !== siteConfig
                    ? `${siteConfig.domain}${currentPath}`
                    : siteConfig.domain;
                console.log(`SLUDGE EXPLODER: Blocking content on ${target}`);
            } else if (style) {
                style.remove();
            }
        }

        // Belt-and-suspenders URL poller: catches SPA navigations that bypass
        // pushState/replaceState patches or only change query/hash state.
        let lastUrlKey = getUrlKey();
        onNavigate = () => {
            lastUrlKey = getUrlKey();
            applyBlocking();
        };
        applyBlocking();

        setInterval(() => {
            const urlKey = getUrlKey();
            if (urlKey !== lastUrlKey) {
                lastUrlKey = urlKey;
                applyBlocking();
            }
        }, 200);

        // Live updates: when the app pushes a new SetConfig, background.js
        // writes it to storage, and every open tab picks it up here without
        // a reload.
        chrome.storage.onChanged.addListener((changes, area) => {
            if (area !== 'local' || !changes[STORAGE_KEY]) return;
            const newConfig = changes[STORAGE_KEY].newValue;
            CONFIG = Array.isArray(newConfig) && newConfig.length > 0 ? newConfig : BUNDLED_CONFIG;
            configHash = sludgeHashConfig(CONFIG);
            siteConfig = selectMostSpecificDomain(currentHost, CONFIG);
            applyBlocking();
        });
    })();

})();
