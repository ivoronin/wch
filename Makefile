.PHONY: build test-all release clean

build:
	zig build -Doptimize=ReleaseSafe

test-all:
	zig build test

release:
	goreleaser release --clean

clean:
	rm -rf .zig-cache zig-out dist
