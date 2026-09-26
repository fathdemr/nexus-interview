# ============================
# Nexus API - Enhanced Makefile
# Includes release-test and release-live targets
# ============================

# ==== Config ====
IMAGE_NAME     ?= 060622564147.dkr.ecr.eu-central-1.amazonaws.com/nexus
TAG            ?= $(shell date +%Y.%m.%d%H%M%S)
PLATFORMS      ?= linux/arm64
BUILD_ARGS     ?=

# Clean tag (remove any whitespace)
TAG := $(shell printf '%s' '$(TAG)' | tr -d '[:space:]')

# ==== Tools ====
DOCKER ?= docker
BUILDX ?= docker buildx

# ==== Helpers ====
GIT_SHA  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "nogit")
DATETAG  := $(shell date +%Y%m%d%H%M)

.PHONY: help login print-tag load release tag-latest clean buildFile swag

help:
	@echo "Nexus API Build Targets:"
	@echo "  make release                   # Build & push with timestamp tag"
	@echo "  make load                      # Load image locally for development"
	@echo "  make tag-latest                # Tag specific version as latest"
	@echo "  make print-tag                 # Print image reference"
	@echo ""
	@echo "Examples:"
	@echo "  make release                   # Build & push with timestamp"
	@echo "  make load                      # Load image locally for testing"


# Build the Go binary
buildFile: swag
	@echo "Stamping version $(TAG) into pkg/config/version.go..."
	@sed -i.bak 's/var Version = "[^"]*"/var Version = "$(TAG)"/' pkg/config/version.go
	@echo "Compiling for Linux arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s" -a -installsuffix cgo -o ./dist/linux/api ./cmd/server
	@mv pkg/config/version.go.bak pkg/config/version.go
	@echo "Version restored to dev"


# ==== SIMPLIFIED TARGETS ====

# Print image tag
print-tag:
	@echo "$(IMAGE_NAME):$(TAG)"

# Load image locally for development
load: buildFile
	$(BUILDX) build \
	  --provenance=mode=max \
	  --sbom=generator=syft \
	  --platform linux/amd64 \
	  -t $(IMAGE_NAME):$(TAG) \
	  --load \
	  $(BUILD_ARGS) .

# Generate Swagger
swag:
	@echo "Generating Swagger docs..."
	swag init --generalInfo cmd/server/swagger.go --parseDependency --parseInternal -q

# Release to repository with latest tag
release: buildFile
	@echo "🚀 Building and releasing: $(IMAGE_NAME):$(TAG)"
	$(BUILDX) build \
	  --platform $(PLATFORMS) \
	  -t $(IMAGE_NAME):$(TAG) \
	  --push \
	  $(BUILD_ARGS) .
	@echo "🏷️ Tagging as latest..."
	$(BUILDX) imagetools create \
	  -t $(IMAGE_NAME):latest \
	  $(IMAGE_NAME):$(TAG)
	@echo "✅ Release completed: $(IMAGE_NAME):$(TAG)"

# Tag specific version as latest
tag-latest:
	@echo "Tagging $(IMAGE_NAME):$(TAG) as latest"
	$(BUILDX) imagetools create \
	  -t $(IMAGE_NAME):latest \
	  $(IMAGE_NAME):$(TAG)
	@echo "✅ Image tagged as latest"

# ==== UTILITIES ====

# Clean Docker system
clean:
	$(DOCKER) system prune -f