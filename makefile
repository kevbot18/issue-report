CC=musl-gcc

all: ticket

ticket: ticket.go
	CC=$(CC) go build --ldflags '-linkmode external -extldflags "-static"' ticket.go

clean: ticket
	rm ticket
