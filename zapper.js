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
    const now = new Date();
    const currentHour = now.getHours();

    const siteConfig = CONFIG.find(entry => matchesDomain(currentHost, entry.domain));

    if (!siteConfig) return;

    let isAllowed = false;
    if (siteConfig.allowWindow) {
        if (currentHour >= siteConfig.allowWindow.start && currentHour < siteConfig.allowWindow.end) {
            isAllowed = true;
        }
    }

    const permablockRules = (siteConfig.permablock_selectors || [])
        .map(selector => `${selector} { display: none !important; visibility: hidden !important; opacity: 0 !important; }`)
        .join(" ");

    const regularRules = (!isAllowed && siteConfig.selectors)
        ? siteConfig.selectors
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

    console.log(`SLUDGE EXPLODER: Blocked content on ${siteConfig.domain}`);

})();