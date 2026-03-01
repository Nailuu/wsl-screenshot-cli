BINARY = wsl-screenshot-cli

.PHONY: build test test-race snapshot release clean

build:
	go build -o $(BINARY) .

test:
	go test -count=1 -v ./...

test-race:
	CGO_ENABLED=1 go test -race -count=1 -v ./...

snapshot:
	goreleaser build --snapshot --clean

release:
	@if [ -z "$(VERSION)" ]; then echo "usage: make release VERSION=0.1.0"; exit 1; fi
	git tag v$(VERSION)
	git push origin main --tags

clean:
	rm -rf $(BINARY) dist/
