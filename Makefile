SHA256SUM_CMD ?= sha256sum

GORELEASER_CMD ?= goreleaser

TAILWIND_DOWNLOAD_URL ?= https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.6/tailwindcss-macos-arm64
TAILWINDCSS_SHA256SUM ?= cf243cf637a99b1f4fecc3f113b3ae810225cc6e0f3d570d44cf8a0ef2e1b29c

GO_FILES := $(shell find . -type f \( -name '*.go' -o -name '*.html' -o -name '*.css' \) -not -name '*_test.go')
HTML_FILES := $(shell find . -type f \( -name '*.html'  -o -name '*.css' \) -not -name 'output.css')
VERSION = $(shell cat VERSION)

_output/dist: $(VERSION_FILE) $(GO_FILES) pkg/static/assets/css/output.css
	$(GORELEASER_CMD) build --single-target --snapshot --clean --output _output/dist/staticreg

.PHONY: clean
clean:
	rm -Rf _output/

_output/deps:
	mkdir -p $@

NO_REBUILD_CSS ?= 0
ifeq ($(NO_REBUILD_CSS),0)
_output/deps/tailwindcss: _output/deps
	curl -o $@ -sLO $(TAILWIND_DOWNLOAD_URL)
	echo "$(TAILWINDCSS_SHA256SUM)  $@"
	chmod +x $@

tools:
.PHONY: deps
deps:
	go mod tidy
	go mod verify


pkg/static/assets/css/output.css: $(HTML_FILES) ./pkg/static/input.css _output/deps/tailwindcss
	_output/deps/tailwindcss ./pkg/static/input.css -o $@
endif

.PHONY: release
release:
	git tag -a "$(VERSION)" -m "$(VERSION)"
	git push origin $(VERSION)

.PHONY: snapshot
snapshot:
	$(GORELEASER_CMD) release --snapshot  --clean
