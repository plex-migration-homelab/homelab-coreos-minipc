FROM golang:1.24-bookworm

# Install all packages as root
RUN apt-get update && apt-get install -y git sudo zsh \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user 'vscode' with passwordless sudo and zsh shell
RUN useradd -m -s /bin/zsh vscode \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers.d/nopasswd \
    && chmod 0440 /etc/sudoers.d/nopasswd

# Create cache directory with correct permissions
RUN mkdir -p /home/vscode/.cache && chown -R vscode:vscode /home/vscode/.cache

# Set ownership of workspace to vscode user
WORKDIR /workspace
RUN chown -R vscode:vscode /workspace

# Switch to non-root user
USER vscode
