SHA256SUM_CMD ?= sha256sum

GORELEASER_CMD ?= goreleaser


 ifeq (, $(shell which $(GORELEASER_CMD)))
 	$(error "goreleaser is not installed. Install it via go install github.com/goreleaser/goreleaser/v2@latest")
 endif

TAILWIND_DOWNLOAD_URL ?= https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.6/tailwindcss-linux-x64
TAILWINDCSS_SHA256SUM ?= 0948afc4cd6b25fa7970cd5336411495d004ecf672e8654b149883e09bb85db5

GO_FILES := $(shell find . -type f \( -name '*.go' -o -name '*.html' -o -name '*.css' \) -not -name '*_test.go')
VERSION = $(shell cat VERSION)

RELEASE_BUILD ?= 0
ifeq ($(RELEASE_BUILD),0)
GORELEASER_BUILD_FLAGS = --single-target --snapshot --clean --output _output/dist/statireg
else
GORELEASER_BUILD_FLAGS = --clean
endif

_output/dist: $(VERSION_FILE) $(GO_FILES) static/css/output.css
	$(GORELEASER_CMD) build $(GORELEASER_BUILD_FLAGS)

.PHONY: clean
clean:
	rm -Rf _output/

_output/deps:
	mkdir -p $@

NO_REBUILD_CSS ?= 0
ifeq ($(NO_REBUILD_CSS),0)
_output/deps/tailwindcss: _output/deps
	curl -o $@ -sLO $(TAILWIND_DOWNLOAD_URL)
	echo "$(TAILWINDCSS_SHA256SUM)  $@" | $(SHA256SUM_CMD) --check
	chmod +x $@


tools:
.PHONY: deps
deps:
	go mod tidy
	go mod verify


static/css/output.css: $(GO_FILES) ./static/css/input.css _output/deps/tailwindcss
	_output/deps/tailwindcss ./static/css/input.css -o $@
endif

.PHONY: release
release:
	git tag -a "v$(VERSION)" -m "v$(VERSION)"
	git push origin v$(VERSION)
	$(GORELEASER_CMD) release --clean --fail-fast
