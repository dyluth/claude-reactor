# Multi-stage Dockerfile for Claude CLI container variants
# Each stage builds upon the previous to create specialized development environments

# =============================================================================
# BASE STAGE: Core system + Node.js + Python + Claude CLI
# =============================================================================
FROM debian:bullseye-slim AS base

# Install comprehensive development dependencies for Claude CLI
# The # bust-cache comment is added to force a re-run of this layer
RUN apt-get update && apt-get install -y \
    # Core system tools # bust-cache
    curl git ca-certificates wget unzip gnupg2 socat sudo \
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
# Detect architecture and set the appropriate kubectl binary URL
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then KUBE_ARCH="arm64"; \
    elif [ "$ARCH" = "amd64" ]; then KUBE_ARCH="amd64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/${KUBE_ARCH}/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install GitHub CLI using the recommended method
RUN type -p curl >/dev/null || (apt-get update && apt-get install curl -y) && \
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
    chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
    apt-get update && \
    apt-get install gh -y

# Install Docker CLI (for connecting to host Docker daemon)
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian bullseye stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null && \
    apt-get update && apt-get install -y docker-ce-cli && rm -rf /var/lib/apt/lists/*

# --- Install Node.js and Claude CLI ---
# Set environment variables for nvm
ENV NVM_DIR=/usr/local/nvm
ENV NODE_VERSION=20.18.0

# Create NVM directory and install nvm
RUN mkdir -p $NVM_DIR && \
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

# Activate nvm and install Node.js and essential Node.js tools as root
# We do this in a single RUN command to ensure it all happens in the same shell context.
RUN . "$NVM_DIR/nvm.sh" && \
    nvm install $NODE_VERSION && \
    nvm use $NODE_VERSION && \
    nvm alias default $NODE_VERSION && \
    npm install -g typescript ts-node eslint prettier

# Add the nvm-installed node and npm to the PATH for all future shell sessions
ENV PATH=$NVM_DIR/versions/node/v$NODE_VERSION/bin:$PATH

# --- Install uv (modern Python package manager) and create Python symlinks ---
RUN curl -LsSf https://astral.sh/uv/install.sh | sh && \
    mv /root/.local/bin/uv /usr/local/bin/uv && \
    mv /root/.local/bin/uvx /usr/local/bin/uvx && \
    ln -sf /usr/bin/python3 /usr/local/bin/python && \
    ln -sf /usr/bin/pip3 /usr/local/bin/pip && \
    rm -rf /root/.local

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

# Copy bashrc configuration to the claude user and clean up uv references
RUN cp /root/.bashrc /home/claude/.bashrc && \
    cp /root/.bash_profile /home/claude/.bash_profile && \
    sed -i '/\$HOME\/\.local\/bin\/env/d' /home/claude/.bashrc && \
    sed -i '/\$HOME\/\.local\/bin\/env/d' /home/claude/.bash_profile && \
    chown claude:claude /home/claude/.bashrc /home/claude/.bash_profile

# Setup npm permissions for claude user and install Claude CLI
RUN chown -R claude:claude $NVM_DIR

USER claude
RUN . "$NVM_DIR/nvm.sh" && \
    nvm use $NODE_VERSION && \
    npm install -g @anthropic-ai/claude-code && \
    echo 'export PATH="$(npm config get prefix)/bin:$PATH"' >> /home/claude/.bashrc

USER root

# Set the working directory for when we connect to the container
WORKDIR /app

# Change ownership of the app directory to claude user
RUN chown -R claude:claude /app

# --- Add and configure the entrypoint for socat proxy ---
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Switch to non-root user
USER claude

# The entrypoint script will now handle the container's main process
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# The default command to run when the container starts.
# This keeps the container alive in detached mode.
CMD ["tail", "-f", "/dev/null"]


# =============================================================================
# GO STAGE: Base + Go toolchain and utilities
# =============================================================================
FROM base AS go

# Switch to root for installations
USER root

# Install Go
ENV GO_VERSION=1.21.6
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then GO_ARCH="arm64"; \
    elif [ "$ARCH" = "amd64" ]; then GO_ARCH="amd64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" | tar -xz -C /usr/local
ENV PATH=/usr/local/go/bin:$PATH

# Install Go development tools
USER root
RUN /usr/local/go/bin/go install golang.org/x/tools/gopls@latest && \
    /usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest && \
    /usr/local/go/bin/go install honnef.co/go/tools/cmd/staticcheck@latest && \
    /usr/local/go/bin/go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Copy Go binaries to system path for all users
RUN cp /root/go/bin/* /usr/local/bin/ 2>/dev/null || true

# Switch back to claude user
USER claude

# =============================================================================
# FULL STAGE: Go + Rust + Java + Database clients
# =============================================================================
FROM go AS full

# Switch to root for installations
USER root

# Install Rust
ENV RUST_VERSION=1.75.0
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain $RUST_VERSION
ENV PATH=/root/.cargo/bin:$PATH

# Install Rust development tools
RUN /root/.cargo/bin/cargo install cargo-edit cargo-watch cargo-audit

# Install Java (OpenJDK 17)
RUN apt-get update && apt-get install -y \
    openjdk-17-jdk \
    maven \
    gradle \
    && rm -rf /var/lib/apt/lists/*

ENV JAVA_HOME=/usr/lib/jvm/java-17-openjdk-arm64

# Install database clients
RUN apt-get update && apt-get install -y \
    mysql-client \
    postgresql-client \
    redis-tools \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Install additional utilities
RUN apt-get update && apt-get install -y \
    tree \
    yq \
    rsync \
    openssl \
    netcat \
    telnet \
    && rm -rf /var/lib/apt/lists/*

# Copy Rust toolchain to system path
RUN cp /root/.cargo/bin/cargo /usr/local/bin/ && \
    cp /root/.cargo/bin/rustc /usr/local/bin/ && \
    cp /root/.cargo/bin/rustup /usr/local/bin/

# Switch back to claude user
USER claude

# =============================================================================
# CLOUD STAGE: Full + Cloud provider CLIs
# =============================================================================
FROM full AS cloud

# Switch to root for installations
USER root

# Install AWS CLI v2
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then AWS_ARCH="aarch64"; \
    elif [ "$ARCH" = "amd64" ]; then AWS_ARCH="x86_64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl "https://awscli.amazonaws.com/awscli-exe-linux-${AWS_ARCH}.zip" -o "awscliv2.zip" && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -rf aws awscliv2.zip

# Install Google Cloud CLI
RUN curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
    echo "deb https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    apt-get update && apt-get install -y google-cloud-cli && \
    rm -rf /var/lib/apt/lists/*

# Install Azure CLI
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash

# Install Terraform
ENV TERRAFORM_VERSION=1.6.6
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then TF_ARCH="arm64"; \
    elif [ "$ARCH" = "amd64" ]; then TF_ARCH="amd64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -fsSL "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_${TF_ARCH}.zip" -o terraform.zip && \
    unzip terraform.zip && \
    mv terraform /usr/local/bin/ && \
    rm terraform.zip

# Switch back to claude user
USER claude

# =============================================================================
# K8S STAGE: Full + Enhanced Kubernetes tools
# =============================================================================
FROM full AS k8s

# Switch to root for installations
USER root

# Install Helm
RUN curl https://baltocdn.com/helm/signing.asc | apt-key add - && \
    echo "deb https://baltocdn.com/helm/stable/debian/ all main" | tee /etc/apt/sources.list.d/helm-stable-debian.list && \
    apt-get update && apt-get install -y helm && \
    rm -rf /var/lib/apt/lists/*

# Install k9s
ENV K9S_VERSION=v0.29.1
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then K9S_ARCH="arm64"; \
    elif [ "$ARCH" = "amd64" ]; then K9S_ARCH="x86_64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -fsSL "https://github.com/derailed/k9s/releases/download/${K9S_VERSION}/k9s_Linux_${K9S_ARCH}.tar.gz" | tar -xz -C /usr/local/bin

# Install kubectx and kubens
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then KUBE_UTILS_ARCH="arm64"; \
    elif [ "$ARCH" = "amd64" ]; then KUBE_UTILS_ARCH="x86_64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -fsSL "https://github.com/ahmetb/kubectx/releases/latest/download/kubectx_linux_${KUBE_UTILS_ARCH}.tar.gz" | tar -xz -C /usr/local/bin kubectx && \
    curl -fsSL "https://github.com/ahmetb/kubectx/releases/latest/download/kubens_linux_${KUBE_UTILS_ARCH}.tar.gz" | tar -xz -C /usr/local/bin kubens

# Install kustomize
RUN curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash && \
    mv kustomize /usr/local/bin/

# Install stern (log tailing)
ENV STERN_VERSION=1.28.0
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then STERN_ARCH="arm64"; \
    elif [ "$ARCH" = "amd64" ]; then STERN_ARCH="amd64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -fsSL "https://github.com/stern/stern/releases/download/v${STERN_VERSION}/stern_${STERN_VERSION}_linux_${STERN_ARCH}.tar.gz" | tar -xz -C /usr/local/bin stern

# Switch back to claude user
USER claude
