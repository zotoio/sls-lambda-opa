build:
	dep ensure
	env GOOS=linux go build -ldflags="-s -w" -o bin/opacheck opacheck/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/gold gold/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/silver silver/main.go