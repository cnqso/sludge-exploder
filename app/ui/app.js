let catalog = [];
let prefs = { selections: {}, customDomains: [], durationMinutes: 60 };
let browsers = [];

async function init() {
    catalog = await getCatalog();
    prefs = await getPrefs();
    if (!prefs.selections) prefs.selections = {};
    if (!prefs.customDomains) prefs.customDomains = [];
    document.getElementById('duration-input').value = prefs.durationMinutes || 60;

    renderCatalog();
    renderCustomDomains();
    renderLockSummary();

    await refreshStatus();
    setInterval(refreshStatus, 1500);

    await refreshLockStatus();
    setInterval(refreshLockStatus, 1000);

    await initStartupToggle();
}

async function initStartupToggle() {
    const checkbox = document.getElementById('startup-checkbox');
    const note = document.getElementById('startup-note');
    // isStartOnLoginEnabled() never errors -- it just reports false on
    // platforms without an implementation (see app/startup_other.go).
    // Attempting to actually toggle it is what surfaces "not implemented"
    // on those platforms, handled in the change listener below.
    checkbox.checked = await isStartOnLoginEnabled();

    checkbox.addEventListener('change', async (e) => {
        const wantEnabled = e.target.checked;
        try {
            if (wantEnabled) {
                await enableStartOnLogin();
            } else {
                await disableStartOnLogin();
            }
            note.textContent = '';
        } catch (err) {
            note.textContent = `Failed: ${err}`;
            e.target.checked = !wantEnabled;
        }
    });
}

function renderCatalog() {
    const root = document.getElementById('catalog');
    root.innerHTML = '';
    for (const app of catalog) {
        const appDiv = document.createElement('div');
        appDiv.className = 'catalog-app';

        const title = document.createElement('h3');
        title.textContent = `${app.label} (${app.domain})`;
        appDiv.appendChild(title);

        for (const part of app.parts) {
            const label = document.createElement('label');
            label.className = 'part-toggle';

            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.checked = !!(prefs.selections[app.id] && prefs.selections[app.id][part.id]);
            checkbox.addEventListener('change', () => {
                if (!prefs.selections[app.id]) prefs.selections[app.id] = {};
                prefs.selections[app.id][part.id] = checkbox.checked;
            });

            label.appendChild(checkbox);
            label.appendChild(document.createTextNode(' ' + part.label));
            appDiv.appendChild(label);
        }
        root.appendChild(appDiv);
    }
}

function renderCustomDomains() {
    const list = document.getElementById('custom-domain-list');
    list.innerHTML = '';
    prefs.customDomains.forEach((cd, index) => {
        const li = document.createElement('li');
        li.textContent = `${cd.domain}: ${cd.selectors.join(', ')} `;

        const removeBtn = document.createElement('button');
        removeBtn.textContent = 'Remove';
        removeBtn.addEventListener('click', () => {
            prefs.customDomains.splice(index, 1);
            renderCustomDomains();
        });

        li.appendChild(removeBtn);
        list.appendChild(li);
    });
}

document.getElementById('add-custom-domain').addEventListener('click', () => {
    const domainInput = document.getElementById('custom-domain-input');
    const selectorsInput = document.getElementById('custom-selectors-input');
    const domain = domainInput.value.trim();
    const selectors = selectorsInput.value.split(',').map(s => s.trim()).filter(Boolean);
    if (!domain || selectors.length === 0) return;

    prefs.customDomains.push({ domain, selectors });
    domainInput.value = '';
    selectorsInput.value = '';
    renderCustomDomains();
});

document.getElementById('save-button').addEventListener('click', async () => {
    prefs.durationMinutes = parseInt(document.getElementById('duration-input').value, 10) || prefs.durationMinutes;
    prefs = await savePrefs(prefs);
    renderLockSummary();
    await refreshStatus();
});

function renderBrowsers() {
    const root = document.getElementById('browser-list');
    root.innerHTML = '';

    if (browsers.length === 0) {
        root.textContent = 'No supported browsers detected on this machine.';
        return;
    }

    for (const b of browsers) {
        const row = document.createElement('div');
        row.className = 'browser-row';

        const status = document.createElement('span');
        status.className = 'browser-status ' + (b.connected ? 'ok' : 'missing');
        status.textContent = b.connected ? '✓' : '✗';
        row.appendChild(status);

        const label = document.createElement('span');
        label.textContent = b.label;
        row.appendChild(label);

        if (b.connected) {
            const heartbeat = document.createElement('span');
            heartbeat.className = 'heartbeat-badge ' + (b.alive ? 'alive' : 'stale');
            heartbeat.textContent = b.alive ? 'ALIVE' : 'MISSING';
            row.appendChild(heartbeat);
        }

        if (!b.hostRegistered) {
            // Auto-registered on app startup (fixed extension IDs, no user
            // input needed) -- a visible button here only means that failed,
            // e.g. a permissions error, so offer a retry.
            const btn = document.createElement('button');
            btn.textContent = 'Retry setup';
            btn.addEventListener('click', async () => {
                await registerBrowserHost(b.key);
                await refreshStatus();
            });
            row.appendChild(btn);
        } else if (!b.connected) {
            const note = document.createElement('span');
            note.className = 'registered-note';
            note.textContent = '(waiting for the extension -- load it unpacked from extension/)';
            row.appendChild(note);
        }

        root.appendChild(row);
    }
}

async function refreshStatus() {
    browsers = await getConnectionStatus();
    renderBrowsers();
    document.getElementById('status-panel').textContent = JSON.stringify(browsers, null, 2);
}

function renderLockSummary() {
    document.getElementById('lock-summary').textContent = prefs.lastLockIntent
        ? `Last lock: ${prefs.lastLockIntent.summary} (confirmed ${prefs.lastLockIntent.confirmedAt})`
        : '';
}

document.getElementById('lock-button').addEventListener('click', () => {
    const duration = parseInt(document.getElementById('duration-input').value, 10) || prefs.durationMinutes;
    const blockedApps = Object.values(prefs.selections).filter(parts => Object.values(parts).some(Boolean)).length;

    document.getElementById('lock-confirm-text').textContent =
        `Blocking ${blockedApps} app(s) for ${duration} minute(s). This starts a real, daemon-enforced lock.`;
    document.getElementById('lock-confirm-dialog').showModal();

    document.getElementById('lock-confirm-yes').onclick = async () => {
        document.getElementById('lock-confirm-dialog').close();
        try {
            prefs.lastLockIntent = await confirmLock(duration);
        } catch (err) {
            alert(`Failed to start lock: ${err}`);
        }
        renderLockSummary();
        await refreshLockStatus();
    };
    document.getElementById('lock-confirm-no').onclick = () => {
        document.getElementById('lock-confirm-dialog').close();
    };
});

async function refreshLockStatus() {
    const status = await getLockStatus();
    renderLockStatus(status);
}

function renderLockStatus(status) {
    const countdownEl = document.getElementById('lock-countdown');
    if (status.state === 'LOCKED' && status.until) {
        const remainingMs = new Date(status.until).getTime() - Date.now();
        countdownEl.textContent = remainingMs > 0
            ? `Locked -- ${formatDuration(remainingMs)} remaining`
            : 'Locked -- expiring...';
    } else if (status.daemonUp) {
        countdownEl.textContent = 'Not locked.';
    } else {
        countdownEl.textContent = 'Daemon not running -- start it with ./bin/daemon';
    }

    const checkbox = document.getElementById('enforcement-checkbox');
    if (document.activeElement !== checkbox) {
        checkbox.checked = status.enforcement;
    }
    document.getElementById('enforcement-state').textContent = status.enforcement ? 'ON' : 'OFF';

    renderAtRisk(status.atRisk || []);
}

function renderAtRisk(atRisk) {
    const banner = document.getElementById('at-risk-banner');
    if (atRisk.length === 0) {
        banner.hidden = true;
        banner.textContent = '';
        return;
    }
    banner.hidden = false;
    banner.textContent = atRisk
        .map(r => `⚠ ${r.label} will close in ${r.graceRemainingSeconds}s unless its extension reconnects`)
        .join(' — ');
}

function formatDuration(ms) {
    const totalSeconds = Math.max(0, Math.round(ms / 1000));
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}m ${seconds}s`;
}

document.getElementById('unlock-button').addEventListener('click', async () => {
    const resultEl = document.getElementById('unlock-result');
    try {
        const status = await attemptUnlock();
        resultEl.textContent = 'Unlocked.';
        renderLockStatus(status);
    } catch (err) {
        resultEl.textContent = `Refused: ${err}`;
    }
});

document.getElementById('enforcement-checkbox').addEventListener('change', async (e) => {
    const enabled = e.target.checked;
    try {
        const status = await setEnforcement(enabled);
        renderLockStatus(status);
    } catch (err) {
        alert(`Failed to change enforcement: ${err}`);
        e.target.checked = !enabled;
    }
});

init();
