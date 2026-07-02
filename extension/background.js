// Service worker. Stage 1 gave it a console for manual testing (still
// available below via sludgeGetStatus()). Stage 2 adds the native-messaging
// client that talks to the app's config bridge (docs/ENFORCEMENT.md §5.1):
//   - sends {type:'hello'} on connect
//   - applies inbound {type:'setConfig'} to chrome.storage.local
//   - answers {type:'getStatus'} and proactively pushes {type:'status'} on
//     any storage change, so the app's status panel updates live
// Native Messaging spawns app/nmhost/main.go, a dumb relay to the app's
// local socket -- see the Stage 2 plan for why it's a relay, not a direct
// link between this script and the app process.
importScripts('config.js', 'hash.js');

const NATIVE_HOST = 'com.sludgeexploder.host'; // must match shared.NativeHostName
const STORAGE_KEY = 'SLUDGE_CONFIG';
const BUNDLED_CONFIG = globalThis.SLUDGE_CONFIG;

let port = null;
let reconnectDelayMs = 1000;
const MAX_RECONNECT_DELAY_MS = 8000;

async function getActiveConfig() {
    const stored = await chrome.storage.local.get(STORAGE_KEY);
    const storedConfig = stored[STORAGE_KEY];
    return Array.isArray(storedConfig) && storedConfig.length > 0 ? storedConfig : BUNDLED_CONFIG;
}

async function computeStatus() {
    const config = await getActiveConfig();
    return {
        type: 'status',
        extId: chrome.runtime.id,
        version: chrome.runtime.getManifest().version,
        configHash: sludgeHashConfig(config),
        rulesActive: config.length,
    };
}

async function handleAppMessage(message) {
    if (!message || typeof message.type !== 'string') return;

    if (message.type === 'setConfig') {
        const rules = Array.isArray(message.rules) ? message.rules : [];
        await chrome.storage.local.set({ [STORAGE_KEY]: rules });
        const status = await computeStatus();
        port?.postMessage({ type: 'setConfigAck', configHash: status.configHash });
        return;
    }

    if (message.type === 'getStatus') {
        port?.postMessage(await computeStatus());
    }
}

function scheduleReconnect() {
    // Plain setTimeout in an MV3 service worker isn't guaranteed to survive
    // suspension past ~30s idle -- fine for this short backoff (caps at 8s),
    // but Stage 3's formal heartbeat should move to chrome.alarms instead.
    setTimeout(connectToApp, reconnectDelayMs);
    reconnectDelayMs = Math.min(reconnectDelayMs * 2, MAX_RECONNECT_DELAY_MS);
}

function connectToApp() {
    try {
        port = chrome.runtime.connectNative(NATIVE_HOST);
    } catch (err) {
        console.warn('SLUDGE EXPLODER: connectNative failed', err);
        port = null;
        scheduleReconnect();
        return;
    }

    port.onMessage.addListener((message) => { handleAppMessage(message); });
    port.onDisconnect.addListener(() => {
        // Expected until the user finishes setup (host not registered yet)
        // or whenever the app isn't running -- reading chrome.runtime.lastError
        // here is required, not optional: Chrome logs an "Unchecked
        // runtime.lastError" warning to the console for every disconnect
        // where nothing reads it, even totally expected ones like this.
        if (chrome.runtime.lastError) {
            console.debug('SLUDGE EXPLODER: native host disconnected:', chrome.runtime.lastError.message);
        }
        port = null;
        scheduleReconnect();
    });

    reconnectDelayMs = 1000;
    port.postMessage({
        type: 'hello',
        extId: chrome.runtime.id,
        version: chrome.runtime.getManifest().version,
    });
}

chrome.storage.onChanged.addListener((changes, area) => {
    if (area !== 'local' || !changes[STORAGE_KEY] || !port) return;
    computeStatus().then((status) => port.postMessage(status));
});

connectToApp();

// --- Stage 1 dev helper, unchanged: call from the service-worker console. ---
globalThis.sludgeGetStatus = async function (tabId) {
    const targetTabId = tabId ?? (await activeTabId());
    return chrome.tabs.sendMessage(targetTabId, { type: 'getStatus' });
};

async function activeTabId() {
    const win = await chrome.windows.getLastFocused({ windowTypes: ['normal'] });
    const [tab] = await chrome.tabs.query({ active: true, windowId: win.id });
    if (!tab) throw new Error('sludgeGetStatus: no active tab found');
    return tab.id;
}
