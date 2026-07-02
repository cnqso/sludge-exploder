// Service worker. Stage 1 gave it a console for manual testing (still
// available below via sludgeGetStatus()). Stage 2 added a native-messaging
// client that talked to the app. Stage 3 points that same channel at the
// daemon instead (docs/ENFORCEMENT.md §5.1) and turns the one-off "hello"
// into a real heartbeat:
//   - sends {type:'heartbeat', extId, version, configHash} every 5s while
//     the port is open
//   - applies {configToApply} from every heartbeat reply to
//     chrome.storage.local -- the daemon always includes this (see
//     daemon/configstore.go for why it's unconditional, not hash-gated)
//   - still applies an inbound {type:'setConfig'} immediately too, for the
//     Stage 2 live-reload feel when the app pushes a change proactively
// Native Messaging spawns app/nmhost/main.go, a dumb relay to the daemon's
// local socket.
importScripts('config.js', 'hash.js');

const NATIVE_HOST = 'com.sludgeexploder.host'; // must match shared.NativeHostName
const STORAGE_KEY = 'SLUDGE_CONFIG';
const BUNDLED_CONFIG = globalThis.SLUDGE_CONFIG;
const HEARTBEAT_INTERVAL_MS = 5000;

let port = null;
let heartbeatTimer = null;
let reconnectDelayMs = 1000;
const MAX_RECONNECT_DELAY_MS = 8000;

async function getActiveConfig() {
    const stored = await chrome.storage.local.get(STORAGE_KEY);
    const storedConfig = stored[STORAGE_KEY];
    return Array.isArray(storedConfig) && storedConfig.length > 0 ? storedConfig : BUNDLED_CONFIG;
}

async function applyConfig(rules) {
    if (!Array.isArray(rules)) return;
    await chrome.storage.local.set({ [STORAGE_KEY]: rules });
}

function handleDaemonMessage(message) {
    if (!message || typeof message.type !== 'string') return;

    if (message.type === 'status') {
        if (Array.isArray(message.configToApply)) {
            applyConfig(message.configToApply);
        }
        return;
    }

    if (message.type === 'setConfig') {
        applyConfig(message.rules);
    }
}

function sendHeartbeat() {
    if (!port) return;
    getActiveConfig().then((config) => {
        port?.postMessage({
            type: 'heartbeat',
            extId: chrome.runtime.id,
            version: chrome.runtime.getManifest().version,
            configHash: sludgeHashConfig(config),
        });
    });
}

function scheduleReconnect() {
    // Plain setTimeout in an MV3 service worker isn't guaranteed to survive
    // suspension past ~30s idle -- fine for this short backoff (caps at 8s),
    // and irrelevant once connected: an open native-messaging port is one
    // of MV3's documented conditions for keeping the service worker alive,
    // so the heartbeat interval below doesn't share this caveat.
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

    port.onMessage.addListener((message) => { handleDaemonMessage(message); });
    port.onDisconnect.addListener(() => {
        // Expected until the daemon is registered/running -- reading
        // chrome.runtime.lastError here is required, not optional: Chrome
        // logs an "Unchecked runtime.lastError" warning to the console for
        // every disconnect where nothing reads it, even expected ones.
        if (chrome.runtime.lastError) {
            console.debug('SLUDGE EXPLODER: daemon disconnected:', chrome.runtime.lastError.message);
        }
        port = null;
        if (heartbeatTimer) {
            clearInterval(heartbeatTimer);
            heartbeatTimer = null;
        }
        scheduleReconnect();
    });

    reconnectDelayMs = 1000;
    sendHeartbeat();
    heartbeatTimer = setInterval(sendHeartbeat, HEARTBEAT_INTERVAL_MS);
}

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
