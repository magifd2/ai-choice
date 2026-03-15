MODULE  := github.com/magifd2/ai-choice
BINARY  := ai-choice
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

DIST_DIR := dist

# Targets for cross-compilation: OS/ARCH pairs
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

.PHONY: build test lint release clean

## build: compile for the current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: run all unit tests
test:
	go test -v -race ./...

## lint: run go vet
lint:
	go vet ./...

## release: cross-compile and package ZIP archives in dist/
release: clean
	@mkdir -p $(DIST_DIR)
	@$(foreach PLATFORM, $(PLATFORMS), \
		$(eval OS   := $(word 1, $(subst /, ,$(PLATFORM)))) \
		$(eval ARCH := $(word 2, $(subst /, ,$(PLATFORM)))) \
		$(eval EXT  := $(if $(filter windows,$(OS)),.exe,)) \
		$(eval NAME := $(BINARY)-$(VERSION)-$(OS)-$(ARCH)$(EXT)) \
		$(eval ZIP  := $(DIST_DIR)/$(BINARY)-$(VERSION)-$(OS)-$(ARCH).zip) \
		GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(NAME) . && \
		cd $(DIST_DIR) && zip $(BINARY)-$(VERSION)-$(OS)-$(ARCH).zip $(NAME) && cd .. && \
		rm $(DIST_DIR)/$(NAME) && \
		echo "  packaged $(ZIP)" ; \
	)

## clean: remove build artifacts
clean:
	rm -rf $(DIST_DIR) $(BINARY)
