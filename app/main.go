// Command app is the Sludge Exploder preference UI: a Go process hosting a
// system webview. Stage 0 just proves the window opens; the real preference
// flow (Stage 2) fills it in.
package main

import (
	"github.com/webview/webview_go"

	_ "github.com/cnqso/sludge-exploder/shared"
)

func main() {
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Sludge Exploder")
	w.SetSize(800, 600, webview.HintNone)
	w.SetHtml("<html><body style='font-family: sans-serif; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0;'><h1>Sludge Exploder</h1></body></html>")
	w.Run()
}
