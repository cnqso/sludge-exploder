// Shared FNV-1a hash used by both the content script and the background
// service worker to compute a deterministic configHash for status
// reporting. Fast and deterministic -- not cryptographic, just change
// detection.
function sludgeHashConfig(config) {
    const str = JSON.stringify(config);
    let hash = 0x811c9dc5;
    for (let i = 0; i < str.length; i++) {
        hash ^= str.charCodeAt(i);
        hash = Math.imul(hash, 0x01000193);
    }
    return (hash >>> 0).toString(16);
}
