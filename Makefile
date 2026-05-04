.PHONY: build test vet fmt install clean

BIN := pe
PKG := ./cmd/pe

build:
	go build -trimpath -o $(BIN) $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

install:
	go install $(PKG)

clean:
	rm -f $(BIN)
