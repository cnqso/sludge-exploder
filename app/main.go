// Command app is the Sludge Exploder preference UI: a Go process hosting a
// system webview. As of Stage 3, the app itself holds no bridge/lock state
// -- it's a thin client of the daemon (see daemonclient.go), which is what
// the extension's Native Messaging heartbeat actually connects to via
// app/nmhost. See docs/DEVELOPMENT_PLAN.md Stage 3.
package main

import (
	_ "embed"
	"log"
	"strings"

	"github.com/webview/webview_go"
)

//go:embed ui/index.html
var uiHTML string

//go:embed ui/app.js
var uiJS string

//go:embed ui/style.css
var uiCSS string

func main() {
	prefs, err := loadPrefs()
	if err != nil {
		log.Printf("sludge-exploder: loading prefs, starting fresh: %v", err)
		prefs = defaultPrefs()
	}

	app := &App{prefs: prefs}
	app.AutoRegisterHosts()

	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Sludge Exploder")
	w.SetSize(900, 720, webview.HintNone)
	app.windowHandle = w.Window() // for focusSelf (focus_darwin.go/focus_windows.go)

	bind := func(name string, f interface{}) {
		if err := w.Bind(name, f); err != nil {
			log.Fatalf("sludge-exploder: binding %s: %v", name, err)
		}
	}
	bind("getCatalog", app.GetCatalog)
	bind("getPrefs", app.GetPrefs)
	bind("savePrefs", app.SavePrefs)
	bind("getDetectedBrowsers", app.GetDetectedBrowsers)
	bind("getConnectionStatus", app.GetConnectionStatus)
	bind("registerBrowserHost", app.RegisterBrowserHost)
	bind("getLockStatus", app.GetLockStatus)
	bind("confirmLock", app.ConfirmLock)
	bind("attemptUnlock", app.AttemptUnlock)
	bind("setEnforcement", app.SetEnforcement)
	bind("enableStartOnLogin", app.EnableStartOnLogin)
	bind("disableStartOnLogin", app.DisableStartOnLogin)
	bind("isStartOnLoginEnabled", app.IsStartOnLoginEnabled)

	html := strings.Replace(uiHTML, "/*STYLE*/", uiCSS, 1)
	html = strings.Replace(html, "/*SCRIPT*/", uiJS, 1)
	w.SetHtml(html)

	w.Run()
}
