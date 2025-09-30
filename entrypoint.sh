#!/bin/sh
set -e

# This script acts as a process supervisor for the container.
# It starts the socat proxy as a background service and then executes
# the main container command, ensuring both run for the container's lifetime.

# The real Docker socket mounted from the host machine.
DOCKER_SOCKET_HOST="/var/run/docker.sock"

# The new proxy socket that will be created and owned by the 'claude' user.
DOCKER_SOCKET_PROXY="/home/claude/docker.sock"

# Function to clean up child processes on exit
cleanup() {
    echo "entrypoint: Received shutdown signal. Cleaning up..."
    # Kill all child processes of this script. Use sudo because socat is now owned by root.
    sudo pkill -P $$
}

# Trap common shutdown signals to ensure cleanup runs
trap 'cleanup' TERM INT

# Check if the host Docker socket is actually mounted and is a socket.
if [ -S "$DOCKER_SOCKET_HOST" ]; then
    echo "entrypoint: Starting socat proxy..."
    # Launch socat as root using sudo to ensure it can connect to the host socket.
    # The UNIX-LISTEN part still creates the new socket with 'claude' user ownership,
    # so the client tools don't need to run as root.
    sudo socat UNIX-LISTEN:${DOCKER_SOCKET_PROXY},fork,user=claude,group=claude,mode=770 "UNIX-CONNECT:${DOCKER_SOCKET_HOST}" &

    # Wait for the proxy socket file to be created by the socat process.
    echo "entrypoint: Waiting for proxy socket to be available at ${DOCKER_SOCKET_PROXY}..."
    timeout=10 # 5 seconds timeout
    while [ ! -S "$DOCKER_SOCKET_PROXY" ]; do
        if [ "$timeout" -eq 0 ]; then
            echo "entrypoint: Timed out waiting for Docker proxy socket." >&2
            exit 1
        fi
        sleep 0.5
        timeout=$((timeout - 1))
    done
    echo "entrypoint: Proxy socket is ready."

    # Export the DOCKER_HOST for any child processes of this script.
    export DOCKER_HOST="unix://${DOCKER_SOCKET_PROXY}"
    
    # Also write DOCKER_HOST to shell config files so it's available in all shell sessions
    echo "export DOCKER_HOST=\"unix://${DOCKER_SOCKET_PROXY}\"" >> /home/claude/.bashrc
    echo "export DOCKER_HOST=\"unix://${DOCKER_SOCKET_PROXY}\"" >> /home/claude/.bash_profile
    echo "entrypoint: Added DOCKER_HOST to shell config files for persistent access"
fi

# Execute the main command passed to the container (e.g., 'tail -f /dev/null')
# in the background and get its PID.
echo "entrypoint: Executing main container command: $@"
"$@" &
main_pid=$!

# Wait for the main command's process to exit.
# This keeps the entrypoint script alive as the main container process (PID 1),
# preventing the socat child process from being orphaned.
wait $main_pid
