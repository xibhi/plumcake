BINARY := plumcake
CMD := ./cmd/plumcake

.PHONY: all build test lint format clean install

all: build

build:
	go build -o $(BINARY) $(CMD)

test:
	go test -v -race ./...

lint:
	go vet ./...

format:
	gofmt -s -w .

clean:
	rm -f $(BINARY)

install: build
	mkdir -p $(HOME)/.local/bin
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)
