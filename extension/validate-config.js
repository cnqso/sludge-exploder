#!/usr/bin/env node

const fs = require('fs');
const vm = require('vm');

const CONFIG_FILE = 'config.js';
const MANIFEST_FILE = 'manifest.json';

const errors = [];
const warnings = [];

function loadConfig() {
    const source = fs.readFileSync(CONFIG_FILE, 'utf8');
    const sandbox = {};
    sandbox.globalThis = sandbox;

    vm.createContext(sandbox);
    vm.runInContext(source, sandbox, { filename: CONFIG_FILE, timeout: 1000 });

    return sandbox.SLUDGE_CONFIG;
}

function addError(location, message) {
    errors.push(`${location}: ${message}`);
}

function addWarning(location, message) {
    warnings.push(`${location}: ${message}`);
}

function isPlainObject(value) {
    return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function validateSelectorArray(value, location, required = false) {
    if (value === undefined && !required) return;

    if (!Array.isArray(value)) {
        addError(location, 'must be an array');
        return;
    }

    value.forEach((selector, index) => {
        if (typeof selector !== 'string' || selector.trim() === '') {
            addError(`${location}[${index}]`, 'must be a non-empty string');
        }
    });
}

function validateAllowWindow(value, location) {
    if (value === null || value === undefined) return;

    if (!isPlainObject(value)) {
        addError(location, 'must be null or an object');
        return;
    }

    const { start, end } = value;
    if (!Number.isInteger(start) || start < 0 || start > 23) {
        addError(`${location}.start`, 'must be an integer from 0 through 23');
    }
    if (!Number.isInteger(end) || end < 0 || end > 24) {
        addError(`${location}.end`, 'must be an integer from 0 through 24');
    }
    if (start === end) {
        addWarning(location, 'start and end are equal, so selectors are never allowed');
    }
}

function validatePathRule(pathRule, location) {
    if (!isPlainObject(pathRule)) {
        addError(location, 'must be an object');
        return;
    }

    if (typeof pathRule.path !== 'string' || !pathRule.path.startsWith('/')) {
        addError(`${location}.path`, 'must be a path string starting with /');
    }

    validateSelectorArray(pathRule.selectors, `${location}.selectors`);
    validateSelectorArray(pathRule.permablock_selectors, `${location}.permablock_selectors`);
    validateAllowWindow(pathRule.allowWindow, `${location}.allowWindow`);
}

function validateConfig(config) {
    if (!Array.isArray(config)) {
        addError('SLUDGE_CONFIG', 'must be an array');
        return;
    }

    const seenDomains = new Map();

    config.forEach((entry, index) => {
        const location = `SLUDGE_CONFIG[${index}]`;

        if (!isPlainObject(entry)) {
            addError(location, 'must be an object');
            return;
        }

        if (typeof entry.domain !== 'string' || entry.domain.trim() === '') {
            addError(`${location}.domain`, 'must be a non-empty string');
        } else {
            const domain = entry.domain.toLowerCase();
            if (domain.includes('://') || domain.includes('/')) {
                addError(`${location}.domain`, 'must be a hostname, not a URL');
            }
            if (domain.startsWith('.')) {
                addError(`${location}.domain`, 'must not start with a dot');
            }
            if (seenDomains.has(domain)) {
                addWarning(`${location}.domain`, `duplicates ${seenDomains.get(domain)}`);
            } else {
                seenDomains.set(domain, `${location}.domain`);
            }
        }

        validateSelectorArray(entry.selectors, `${location}.selectors`);
        validateSelectorArray(entry.permablock_selectors, `${location}.permablock_selectors`);
        validateAllowWindow(entry.allowWindow, `${location}.allowWindow`);

        if (entry.paths !== undefined) {
            if (!Array.isArray(entry.paths)) {
                addError(`${location}.paths`, 'must be an array');
            } else {
                const seenPaths = new Map();
                entry.paths.forEach((pathRule, pathIndex) => {
                    const pathLocation = `${location}.paths[${pathIndex}]`;
                    validatePathRule(pathRule, pathLocation);

                    if (isPlainObject(pathRule) && typeof pathRule.path === 'string') {
                        if (seenPaths.has(pathRule.path)) {
                            addWarning(`${pathLocation}.path`, `duplicates ${seenPaths.get(pathRule.path)}`);
                        } else {
                            seenPaths.set(pathRule.path, `${pathLocation}.path`);
                        }
                    }
                });
            }
        }
    });
}

function validateManifest() {
    const manifest = JSON.parse(fs.readFileSync(MANIFEST_FILE, 'utf8'));
    const scripts = manifest.content_scripts && manifest.content_scripts[0] && manifest.content_scripts[0].js;

    if (!Array.isArray(scripts)) {
        addError('manifest.content_scripts[0].js', 'must be an array');
        return;
    }

    const configIndex = scripts.indexOf(CONFIG_FILE);
    const zapperIndex = scripts.indexOf('zapper.js');

    if (configIndex === -1) addError('manifest.content_scripts[0].js', `must include ${CONFIG_FILE}`);
    if (zapperIndex === -1) addError('manifest.content_scripts[0].js', 'must include zapper.js');
    if (configIndex !== -1 && zapperIndex !== -1 && configIndex > zapperIndex) {
        addError('manifest.content_scripts[0].js', `${CONFIG_FILE} must load before zapper.js`);
    }

    const resources = manifest.web_accessible_resources || [];
    const exposesConfig = resources.some(entry => (
        Array.isArray(entry.resources) &&
        entry.resources.some(resource => resource === CONFIG_FILE || resource === 'config.json')
    ));

    if (exposesConfig) {
        addWarning('manifest.web_accessible_resources', 'config is exposed to pages');
    }
}

try {
    validateConfig(loadConfig());
    validateManifest();
} catch (error) {
    addError('validator', error.message);
}

warnings.forEach(warning => console.warn(`warning: ${warning}`));

if (errors.length > 0) {
    errors.forEach(error => console.error(`error: ${error}`));
    process.exit(1);
}

console.log(`Validated ${CONFIG_FILE}`);
