FROM golang:1.23-bookworm

# Install all packages as root
RUN apt-get update && apt-get install -y git sudo zsh fzf \
    && rm -rf /var/lib/apt/lists/*

# Install zoxide
RUN curl -sS https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | bash

# Create non-root user with passwordless sudo and zsh shell
RUN useradd -m -s /bin/zsh vscode \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers.d/nopasswd \
    && chmod 0440 /etc/sudoers.d/nopasswd

# Give vscode user ownership of all Go directories
RUN chown -R vscode:vscode /go

# Create and own cache directory
RUN mkdir -p /home/vscode/.cache && chown -R vscode:vscode /home/vscode/.cache

# Set ownership of workspace
WORKDIR /workspace
RUN chown -R vscode:vscode /workspace

USER vscode
