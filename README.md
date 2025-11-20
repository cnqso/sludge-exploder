# Sludge Exploder

Brute, barbaric, unsophisticated algo blocker for Chrome.

Block sites, feeds, and individual elements. Options available for timed breaks.

Requires manual config for maximum friction. As long as you don't press the "disable" button you'll be good. And you wouldn't do that, right?

## Why

Because I'm an ADDICT! I needed a very configurable site block/modification routine, and none of the available extensions were ideal for me.

## Why distribute through github instead of through the extensions store?

Because it's important that the extension is dictated by ugly, temperamental JSON so that you aren't constantly tempted to modify your carefully crafted configuration screen. Your only job is to NOT hit the "disable" button. How hard could that possibly be?

## Setup
1. Download the directory wherever you desire (I'd advise against Desktop or Downloads)
2. Modify the "config.json" if you'd like (Comes preloaded with an *omakase* feed block schedule that you're free to try)
3. Choose the instructions that match your browser:

### Chrome / Edge (Chromium)
1. Go to [chrome://extensions/](chrome://extensions/)
2. Enable "developer mode" in the top-right, then hit "Load unpacked" in the top left
3. Choose the "Sludge Exploder" folder you just downloaded
4. Never, ever hit the "disable button"
5. (Optional) Hit "Details" and enable "Allow in Incognito"

### Firefox
1. Go to `about:debugging#/runtime/this-firefox`
2. Click "Load Temporary Add-on..."
3. Select any file inside the "Sludge Exploder" folder (Firefox will ingest the whole directory)
4. Firefox will unload temporary add-ons on restart; when you're ready to publish permanently, package the folder and load it via "about:addons" or sign it through Mozilla's Add-on Developer Hub.
5. Same rules apply: avoid the disable button.

If you ever modify the config.json, make sure to reload the extension (Chrome: hit the refresh icon next to the disable toggle; Firefox: click "Reload" in `about:debugging`). 

