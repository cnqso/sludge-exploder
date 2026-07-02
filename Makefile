.PHONY: build run-app run-daemon clean dev dev-stop dev-reset dev-status dev-logs

build:
	go build -o bin/app ./app
	go build -o bin/nmhost ./app/nmhost
	go build -o bin/daemon ./daemon

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
