CC=musl-gcc

all: ticket

ticket: ticket.go transactions.go
	CC=$(CC) go build --ldflags '-linkmode external -extldflags "-static"'

clean: ticket
	rm ticket
