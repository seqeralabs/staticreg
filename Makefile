GO ?= go
GO_BUILD_CMD ?= CGO_ENABLED=0 $(GO) build

SHA256SUM_CMD ?= sha256sum

TAILWIND_DOWNLOAD_URL ?= https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.6/tailwindcss-linux-x64
TAILWINDCSS_SHA256SUM ?= 0948afc4cd6b25fa7970cd5336411495d004ecf672e8654b149883e09bb85db5

DEBUG ?= 0
ifeq ($(DEBUG),0)
# for release: remove symbol table and remove DWARF debugging info
BUILD_FLAGS = -ldflags '-s -w'
else
# for debugging: disable function calls inlining and compiler optimizations
BUILD_FLAGS = -gcflags '-N -l'
endif

GO_FILES := $(shell find . -type f \( -name '*.go' -o -name '*.html' -o -name '*.css' \) -not -name '*_test.go')

_output/bin/staticreg: $(GO_FILES)
	$(GO_BUILD_CMD) $(BUILD_FLAGS) -o $@ .

clean:
	rm -Rf _output

_output/deps:
	mkdir -p $@

_output/deps/tailwindcss: _output/deps
	curl -o $@ -sLO $(TAILWIND_DOWNLOAD_URL)
	echo "$(TAILWINDCSS_SHA256SUM)  $@" | $(SHA256SUM_CMD) --check
	chmod +x $@


.PHONY: deps
deps: _output/deps/tailwindcss
	go mod tidy
	go mod verify
