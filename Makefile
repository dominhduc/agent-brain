.PHONY: build install test clean release

build:
	go build -buildvcs=false -o bin/brain ./cmd/brain

install: build
	cp bin/brain /usr/local/bin/brain

test:
	go test ./...

clean:
	rm -rf bin/

release:
	goreleaser release --clean
