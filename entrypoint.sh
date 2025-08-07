#!/bin/sh
set -e

# This script acts as an entrypoint for the container.
# It sets up a proxy for the Docker socket to allow a non-root user
# to access it securely, which is necessary on macOS.

# The real Docker socket mounted from the host machine.
DOCKER_SOCKET_HOST="/var/run/docker.sock"

# The new proxy socket that will be created and owned by the 'claude' user.
DOCKER_SOCKET_PROXY="/home/claude/docker.sock"

# Check if the host Docker socket is actually mounted and is a socket.
if [ -S "$DOCKER_SOCKET_HOST" ]; then
    # Use socat to create a proxy.
    # It listens on the new proxy socket and forwards all connections
    # to the real host socket.
    # The 'fork' option handles multiple connections.
    # The user, group, and mode options ensure the new socket has the correct permissions.
    socat UNIX-LISTEN:${DOCKER_SOCKET_PROXY},fork,user=claude,group=claude,mode=770 "UNIX-CONNECT:${DOCKER_SOCKET_HOST}" &
    
    # Export the DOCKER_HOST environment variable.
    # This tells the Docker CLI and other tools to use our new proxy socket
    # instead of the default one.
    export DOCKER_HOST="unix://${DOCKER_SOCKET_PROXY}"
fi

# Execute the main command passed to the container (e.g., 'bash-with-prompt' or 'claude').
# '$@' represents all the arguments passed to this script.
exec "$@"