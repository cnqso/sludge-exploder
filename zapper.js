(async function () {
    let CONFIG;
    try {
        const configUrl = chrome.runtime.getURL('config.json');
        const response = await fetch(configUrl);
        CONFIG = await response.json();
    } catch (error) {
        console.error('SLUDE EXPLODER: Failed to load config.json', error);
        return;
    }


    const currentHost = window.location.hostname;
    const now = new Date();
    const currentHour = now.getHours();

    const siteConfig = CONFIG.find(entry => currentHost.includes(entry.domain));

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

    console.log(`SLUDE EXPLODER: Blocked content on ${siteConfig.domain}`);

})();