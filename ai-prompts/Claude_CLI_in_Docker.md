## **Claude CLI in Docker (with kubernetes access)**

This project brief is ready. It is structured to be clear for a human user (you) and unambiguous for an AI agent to execute.

* **For the user:** The "Connecting to the Container" section acts as a clear, step-by-step user guide, explaining the two different workflows (manual vs. VS Code) and the two different authentication methods (API key vs. UI login). It now also includes instructions for enabling Kubernetes access.  
* **For an AI agent:** The instructions are explicit, providing the exact file structure and content. The comments within the code blocks explain the *intent* behind each major decision, which helps the AI generate correct and robust code.

## **Project context**

The primary objective is to create a secure, isolated sandbox environment using Docker. This environment will serve as a primary, interactive instance of Claude, allowing a developer to connect and use the claude command-line tool directly.

This setup ensures that all interactions are contained and cannot affect the host machine (your M1 MacBook Pro). It supports authentication via a direct API key or through an interactive web UI login, and **it can connect to a local Kubernetes cluster running in Docker Desktop.**

## **Primary goal**

To instruct Claude to generate a complete, runnable project that includes:

1. A **Dockerfile** to define a secure, ARM64-compatible container environment with the official claude CLI and kubectl tools installed.  
2. A **devcontainer.json** file to enable a seamless, one-click connection and development experience using Visual Studio Code, with clear options for authentication and Kubernetes access.

## **Instructions for Claude**

Please generate the following files based on the project structure and specifications below.

### **1\. Project structure**

Create the following file and directory structure:

.  
├── .devcontainer/  
│   └── devcontainer.json  
└── Dockerfile

### **2\. The Dockerfile**

Create a Dockerfile to build the container image. It must use an ARM64-compatible base image, install Node.js (via nvm), the claude-cli, and the kubectl CLI.

\# Use an official Debian image compatible with ARM64 architecture (M1 Macs)  
\# It's a good neutral base for installing development tools like nvm.  
FROM debian:bullseye-slim

\# Install dependencies required for nvm, kubectl, and general development  
RUN apt-get update && apt-get install \-y curl git ca-certificates && rm \-rf /var/lib/apt/lists/\*

\# \--- Install kubectl \---  
RUN curl \-LO "https://dl.k8s.io/release/$(curl \-L \-s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl" && \\  
    install \-o root \-g root \-m 0755 kubectl /usr/local/bin/kubectl

\# \--- Install Node.js and Claude CLI \---  
\# Set environment variables for nvm  
ENV NVM\_DIR /usr/local/nvm  
ENV NODE\_VERSION 22.17.1

\# Install nvm  
RUN curl \-o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash

\# Activate nvm and install Node.js, then the Claude CLI  
\# We do this in a single RUN command to ensure it all happens in the same shell context.  
RUN . "$NVM\_DIR/nvm.sh" && \\  
    nvm install $NODE\_VERSION && \\  
    nvm use $NODE\_VERSION && \\  
    nvm alias default $NODE\_VERSION && \\  
    npm install \-g @anthropic-ai/claude-cli

\# Add the nvm-installed node and npm to the PATH for all future shell sessions  
ENV PATH $NVM\_DIR/versions/node/v$NODE\_VERSION/bin:$PATH

\# Set the working directory for when we connect to the container  
WORKDIR /app

\# This command will be executed when the container starts.  
\# It keeps the container running so we can connect to it.  
CMD \["tail", "-f", "/dev/null"\]

### **3\. The Dev Container Configuration (.devcontainer/devcontainer.json)**

This file enables the seamless integration with VS Code. It now includes mount instructions for the Kubernetes config and the Kubernetes extension for VS Code.

{  
	"name": "Claude Interactive Environment",  
	"dockerFile": "../Dockerfile",

	// This tells VS Code which extensions to install \*inside\* the container.  
	"customizations": {  
		"vscode": {  
			"extensions": \[  
				"dbaeumer.vscode-eslint",  
				"ms-kubernetes-tools.vscode-kubernetes-tools"  
			\]  
		}  
	},

	// \--- OPTIONAL: For API Key Authentication \---  
	// This block passes your local environment file into the container.  
	// The claude CLI will automatically detect and use the ANTHROPIC\_API\_KEY from this file.  
	// To generate a configuration for interactive UI login instead, this "runArgs" block should be removed.  
	"runArgs": \[  
		"--env-file",  
		"${localEnv:HOME}/.env"  
	\],

	// Mounts required directories into the container.  
	"mounts": \[  
		// Mount the project directory itself.  
		"source=${localWorkspaceFolder},target=/app,type=bind,consistency=cached",  
		// Mount the local Kubernetes config directory to enable kubectl access from inside the container.  
		"source=${localEnv:HOME}/.kube,target=/root/.kube,type=bind,consistency=cached"  
	\]  
}

**Note for user:** For API key authentication to work, you must create a file at \~/.env on your Mac and add the line ANTHROPIC\_API\_KEY=your\_key\_here.

## **Connecting to the container: Options for the user**

Once the files are created, here are the two primary ways to connect to and work within the container.

### **Option A: The direct approach (docker exec)**

This is the classic, manual way to interact with a running container.

1. **Build the image:** docker build \-t claude-reactor .  
2. **Run the container in the background:**  
   * **For API Key Authentication:**  
     docker run \-d \--name claude-agent \-v "$(pwd)":/app \-v "${HOME}/.kube:/root/.kube" \--env-file \~/.env claude-reactor

   * **For Interactive UI Login:**  
     docker run \-d \--name claude-agent \-v "$(pwd)":/app \-v "${HOME}/.kube:/root/.kube" claude-reactor

3. **Connect with a shell:** docker exec \-it claude-agent /bin/bash  
4. **Run Commands:** Inside the container's shell, you can now run both claude and kubectl (e.g., kubectl get pods).

### **Option B: The seamless approach (VS Code Dev Containers)**

This is the **highly recommended** method for a fluid development experience.

1. **Prerequisites:** Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) and the [Visual Studio Code Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers).  
2. **Choose Authentication Method:**  
   * **For API Key Authentication:** Ensure the "runArgs" block is present in .devcontainer/devcontainer.json and your \~/.env file is set up.  
   * **For Interactive UI Login:** Before building, remove or comment out the "runArgs" block in .devcontainer/devcontainer.json.  
3. **Open the Project:** Open the project folder in VS Code.  
4. **Reopen in Container:** A pop-up will appear in the bottom-right corner asking if you want to "Reopen in Container". Click it. (If it doesn't appear, open the Command Palette and run Dev Containers: Reopen in Container).  
5. **Access Tools:** Simply open a new VS Code terminal (Terminal \> New Terminal). You can now run both the claude command and the kubectl command. The VS Code Kubernetes extension will also be active, giving you UI-based cluster management.