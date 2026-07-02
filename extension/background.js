// Minimal service worker. Its main job right now is giving Stage 1 a
// console to work from (chrome://extensions -> "service worker" link):
// write SLUDGE_CONFIG to chrome.storage.local from there, and call
// sludgeGetStatus() to query a tab's content script over messaging.
// The daemon-facing heartbeat relay (docs/ENFORCEMENT.md §5.1) replaces
// this file's role starting in Stage 3.

globalThis.sludgeGetStatus = async function (tabId) {
    const targetTabId = tabId ?? (await activeTabId());
    return chrome.tabs.sendMessage(targetTabId, { type: 'getStatus' });
};

async function activeTabId() {
    // Called from the service worker console. `currentWindow`/`lastFocusedWindow`
    // can resolve to the devtools inspector window itself (which has no
    // tabs), since opening the console just gave it focus. Ask explicitly
    // for the last focused *normal* browser window instead.
    const win = await chrome.windows.getLastFocused({ windowTypes: ['normal'] });
    const [tab] = await chrome.tabs.query({ active: true, windowId: win.id });
    if (!tab) throw new Error('sludgeGetStatus: no active tab found');
    return tab.id;
}
