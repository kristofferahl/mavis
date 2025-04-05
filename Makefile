PKG = github.com/kristofferahl/mavis
COMMIT = $$(git describe --tags --always)
BUILD_LDFLAGS = -X $(PKG)/internal/pkg/version.Commit=$(COMMIT)

default: test

test:
	go test ./... -coverprofile=coverage.out -covermode=count

build:
	@echo "Building mavis"
	mkdir -p ./dist/
	go build -o ./dist/ -ldflags="$(BUILD_LDFLAGS)" .

build-all:
	@echo "Building all platforms"
	mkdir -p ./dist/
	goreleaser build --snapshot --clean

prerelease: build-all
	@test $${VER?Environment variable VER is required}
	git pull origin main --tag
	git tag
	git tag ${VER}

release:
	@test $${GITHUB_TOKEN?Environment variable GITHUB_TOKEN is required}
	git push --tags
	goreleaser release --clean

.PHONY: default test
