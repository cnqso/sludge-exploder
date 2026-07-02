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
        `Blocking ${blockedApps} app(s) for ${duration} minute(s). This is a preview -- nothing is enforced yet.`;
    document.getElementById('lock-confirm-dialog').showModal();

    document.getElementById('lock-confirm-yes').onclick = async () => {
        document.getElementById('lock-confirm-dialog').close();
        prefs.lastLockIntent = await confirmLock(duration);
        renderLockSummary();
    };
    document.getElementById('lock-confirm-no').onclick = () => {
        document.getElementById('lock-confirm-dialog').close();
    };
});

init();
