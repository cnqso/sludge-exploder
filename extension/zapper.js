(function () {
    const CONFIG = globalThis.SLUDGE_CONFIG;

    if (!Array.isArray(CONFIG)) {
        console.error('SLUDGE EXPLODER: SLUDGE_CONFIG not found.');
        return;
    }

    // Patch the history API immediately. This catches same-world navigation
    // calls; the URL poller below covers frameworks that bypass the patch.
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

    const currentHost = window.location.hostname.toLowerCase();
    const siteConfig = selectMostSpecificDomain(currentHost, CONFIG);

    if (!siteConfig) return;

    function applyBlocking() {
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

        let style = document.getElementById('zapper-block-style');
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

})();
