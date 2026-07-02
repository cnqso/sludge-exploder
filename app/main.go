// Command app is the Sludge Exploder preference UI: a Go process hosting a
// system webview, plus a local socket server that the Native Messaging
// relay (app/nmhost) connects to so it can push config into the extension.
// See the Stage 2 plan in docs/DEVELOPMENT_PLAN.md.
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

	socket := NewSocketServer()
	if err := socket.Start(); err != nil {
		log.Fatalf("sludge-exploder: starting local socket server: %v", err)
	}
	defer socket.Stop()

	app := &App{prefs: prefs, socket: socket}
	app.AutoRegisterHosts()

	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Sludge Exploder")
	w.SetSize(900, 720, webview.HintNone)

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
	bind("confirmLock", app.ConfirmLock)

	html := strings.Replace(uiHTML, "/*STYLE*/", uiCSS, 1)
	html = strings.Replace(html, "/*SCRIPT*/", uiJS, 1)
	w.SetHtml(html)

	w.Run()
}
