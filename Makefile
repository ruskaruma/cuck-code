BINARY  := cuck
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GOBUILD := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"
PREFIX  ?= $(HOME)/.local

TARGETS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: build test dist npm install demo clean $(TARGETS)

build:
	$(GOBUILD) -o $(BINARY) ./cmd/cuck

test:
	go vet ./...
	go test ./...

dist: $(TARGETS)

$(TARGETS):
	GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@)) \
		$(GOBUILD) -o dist/$(BINARY)-$(word 1,$(subst /, ,$@))-$(word 2,$(subst /, ,$@))$(if $(findstring windows,$@),.exe) ./cmd/cuck

# Assembles the npm packages in packaging/npm/out (needs Node). VERSION must be semver, e.g. make npm VERSION=v0.1.0
npm: dist
	node packaging/npm/build.mjs $(VERSION)

install: build
	install -d $(PREFIX)/bin
	install -m 0755 $(BINARY) $(PREFIX)/bin/$(BINARY)

# Regenerates the README animation and screenshots (needs no terminal).
SHOTS := 1-she-is-waiting:0.3 2-agent-walks-in:2.8 3-take-me-to-heaven:4.6 4-the-kiss:5.5 5-look-back:6.6 6-wink:7.0
demo:
	go run ./cmd/preview -gif -fps 12 -w 84 -h 26 -o docs/assets/demo.gif
	@for s in $(SHOTS); do go run ./cmd/preview -t $${s#*:} -w 100 -h 32 -o docs/assets/screenshots/$${s%%:*}.png; done
	go run ./cmd/preview -t 7.4 -fx 0.35 -w 100 -h 32 -o docs/assets/screenshots/7-flash.png

clean:
	rm -rf dist packaging/npm/out $(BINARY)
