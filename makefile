CC=musl-gcc

all: alert

alert: alert.go
	CC=$(CC) go build --ldflags '-linkmode external -extldflags "-static"' alert.go

clean: alert
	rm alert