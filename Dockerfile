# Use an official Debian image compatible with ARM64 architecture (M1 Macs)
# It's a good neutral base for installing development tools like nvm.
FROM debian:bullseye-slim

# Install dependencies required for nvm, kubectl, and general development
RUN apt-get update && apt-get install -y curl git ca-certificates && rm -rf /var/lib/apt/lists/*

# --- Install kubectl ---
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# --- Install Node.js and Claude CLI ---
# Set environment variables for nvm
ENV NVM_DIR=/usr/local/nvm
ENV NODE_VERSION=22.17.0

# Create NVM directory and install nvm
RUN mkdir -p $NVM_DIR && \
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

# Activate nvm and install Node.js, then the Claude CLI
# We do this in a single RUN command to ensure it all happens in the same shell context.
RUN . "$NVM_DIR/nvm.sh" && \
    nvm install $NODE_VERSION && \
    nvm use $NODE_VERSION && \
    nvm alias default $NODE_VERSION && \
    npm install -g @anthropic-ai/claude-code

# Add the nvm-installed node and npm to the PATH for all future shell sessions
ENV PATH=$NVM_DIR/versions/node/v$NODE_VERSION/bin:$PATH

# Set the working directory for when we connect to the container
WORKDIR /app

# This command will be executed when the container starts.
# It keeps the container running so we can connect to it.
CMD ["tail", "-f", "/dev/null"]