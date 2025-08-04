# Use an official Debian image compatible with ARM64 architecture (M1 Macs)
# It's a good neutral base for installing development tools like nvm.
FROM debian:bullseye-slim

# Install comprehensive development dependencies for Claude CLI
RUN apt-get update && apt-get install -y \
    # Core system tools
    curl git ca-certificates wget unzip gnupg2 \
    # Essential CLI tools for Claude
    ripgrep jq fzf nano vim less procps htop \
    # Build tools and compilers
    build-essential python3 python3-pip \
    # Shell and process tools
    shellcheck man-db \
    && rm -rf /var/lib/apt/lists/*

# --- Install git-aware-prompt ---
RUN git clone https://github.com/jimeh/git-aware-prompt.git /usr/local/git-aware-prompt

# --- Install kubectl and GitHub CLI ---
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
    chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
    apt-get update && apt-get install -y gh && rm -rf /var/lib/apt/lists/*

# Install Docker CLI (for connecting to host Docker daemon)
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian bullseye stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null && \
    apt-get update && apt-get install -y docker-ce-cli && rm -rf /var/lib/apt/lists/*

# --- Install Node.js and Claude CLI ---
# Set environment variables for nvm
ENV NVM_DIR=/usr/local/nvm
ENV NODE_VERSION=22.17.0

# Create NVM directory and install nvm
RUN mkdir -p $NVM_DIR && \
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

# Activate nvm and install Node.js, then the Claude CLI and essential Node.js tools
# We do this in a single RUN command to ensure it all happens in the same shell context.
RUN . "$NVM_DIR/nvm.sh" && \
    nvm install $NODE_VERSION && \
    nvm use $NODE_VERSION && \
    nvm alias default $NODE_VERSION && \
    npm install -g @anthropic-ai/claude-code typescript ts-node eslint prettier

# Add the nvm-installed node and npm to the PATH for all future shell sessions
ENV PATH=$NVM_DIR/versions/node/v$NODE_VERSION/bin:$PATH

# --- Configure git-aware-prompt ---
# Add to both .bashrc and .bash_profile to ensure it loads in all contexts
RUN echo 'export GITAWAREPROMPT=/usr/local/git-aware-prompt' >> /root/.bashrc && \
    echo 'source "${GITAWAREPROMPT}/main.sh"' >> /root/.bashrc && \
    echo 'export PS1="\u@\h \W \[\$txtcyn\]\$git_branch\[\$txtred\]\$git_dirty\[\$txtrst\]\$ "' >> /root/.bashrc && \
    cp /root/.bashrc /root/.bash_profile

# Create a script to ensure git-aware-prompt is always available for claude user
RUN echo '#!/bin/bash' > /usr/local/bin/bash-with-prompt && \
    echo 'export GITAWAREPROMPT=/usr/local/git-aware-prompt' >> /usr/local/bin/bash-with-prompt && \
    echo 'source "${GITAWAREPROMPT}/main.sh"' >> /usr/local/bin/bash-with-prompt && \
    echo 'export PS1="\u@\h \W \[\$txtcyn\]\$git_branch\[\$txtred\]\$git_dirty\[\$txtrst\]\$ "' >> /usr/local/bin/bash-with-prompt && \
    echo 'exec bash "$@"' >> /usr/local/bin/bash-with-prompt && \
    chmod +x /usr/local/bin/bash-with-prompt

# Create a non-root user for Claude CLI (required for --dangerously-skip-permissions)
RUN useradd -m -s /bin/bash claude && \
    usermod -aG sudo claude && \
    echo 'claude ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# Copy bashrc configuration to the claude user
RUN cp /root/.bashrc /home/claude/.bashrc && \
    cp /root/.bash_profile /home/claude/.bash_profile && \
    chown claude:claude /home/claude/.bashrc /home/claude/.bash_profile

# Set the working directory for when we connect to the container
WORKDIR /app

# Change ownership of the app directory to claude user
RUN chown -R claude:claude /app

# Switch to non-root user
USER claude

# This command will be executed when the container starts.
# It keeps the container running so we can connect to it.
CMD ["tail", "-f", "/dev/null"]