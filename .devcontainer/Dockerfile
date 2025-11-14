FROM golang:1.24-bookworm

# Git is required for go mod download (dependencies from GitHub)
# and for the Makefile which embeds git commit hash in the binary
RUN apt-get update && apt-get install -y git \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user 'vscode' with passwordless sudo, set up home directory
RUN useradd -m -s /bin/bash vscode \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers.d/nopasswd \
    && chmod 0440 /etc/sudoers.d/nopasswd

# Set ownership of workspace to vscode user (if applicable)
WORKDIR /workspace
RUN chown -R vscode:vscode /workspace

# Switch to non-root user
USER vscode
