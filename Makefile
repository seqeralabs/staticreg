GO ?= go

GO_BUILD_CMD ?= CGO_ENABLED=0 $(GO) build

DEBUG ?= 0
ifeq ($(DEBUG),0)
# for release: remove symbol table and remove DWARF debugging info
BUILD_FLAGS = -ldflags '-s -w'
else
# for debugging: disable function calls inlining and compiler optimizations
BUILD_FLAGS = -gcflags '-N -l'
endif

GO_FILES := $(shell find . -type f -name '*.go' -not -name '*_test.go')

_output/bin/staticreg: $(GO_FILES)
	$(GO_BUILD_CMD) $(BUILD_FLAGS) -o $@ .

clean:
	rm -Rf _output
