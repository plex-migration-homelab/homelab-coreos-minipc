FROM golang:1.24-bookworm

# Git is required for go mod download (dependencies from GitHub)
# and for the Makefile which embeds git commit hash in the binary
RUN apt-get update && apt-get install -y git \
    && rm -rf /var/lib/apt/lists/*
