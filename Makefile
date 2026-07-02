.PHONY: build run-app run-daemon clean dev dev-stop dev-reset dev-status dev-logs

# go build -o bin/app (no extension) does NOT get .exe auto-appended on
# Windows -- it produces a literal extensionless file, which silently
# breaks nmhostPath()/daemonPath() (app/hostinstall.go, app/startup_windows.go),
# both of which look for "nmhost.exe"/"daemon.exe" specifically. `go env
# GOEXE` is the portable way to get the right suffix (".exe" on Windows,
# "" everywhere else) for whichever platform this is actually running on.
GOEXE := $(shell go env GOEXE)

build:
	go build -o bin/app$(GOEXE) ./app
	go build -o bin/nmhost$(GOEXE) ./app/nmhost
	go build -o bin/daemon$(GOEXE) ./daemon

run-app:
	go run ./app

run-daemon:
	go run ./daemon

clean:
	rm -rf bin

# Dev helper: builds + starts daemon and app together in the background.
# See ./dev.sh for the full command list (stop/reset/wipe-all/status/logs).
dev:
	./dev.sh start

dev-stop:
	./dev.sh stop

dev-reset:
	./dev.sh reset

dev-status:
	./dev.sh status

dev-logs:
	./dev.sh logs
