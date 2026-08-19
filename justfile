set shell := ["bash", "-c"]

# List available recipes
default:
    @just --list

# Compile the CLI binary into bin/blog-engine
build:
    @mkdir -p bin
    go build -o bin/blog-engine ./cmd/blog-engine

# Build the static site into the configured output directory (dist/ by default)
generate: build
    ./bin/blog-engine build

# Run the development server with live reload. Usage: just serve [port]
serve port="3000": build
    ./bin/blog-engine serve --port {{ port }}

# Generate missing or stale article embeddings (requires OPENAI_API_KEY)
embed: build
    ./bin/blog-engine embed

# Preview embedding work and estimated cost without network access
embed-dry-run: build
    ./bin/blog-engine embed --dry-run

# Verify that article frontmatter embeddings are complete and current
embed-check: build
    ./bin/blog-engine embed --check

# Re-generate every eligible article embedding (requires OPENAI_API_KEY)
embed-force: build
    ./bin/blog-engine embed --force

# Run the Go test suite
test:
    go test -v ./...

# Remove local build output, generated site files, and the build cache
clean:
    rm -rf dist/ bin/ .cache/ content/embeddings.json
