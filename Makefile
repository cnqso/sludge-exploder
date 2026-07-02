.PHONY: build run-app run-daemon clean

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
