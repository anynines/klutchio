#! /usr/bin/env bash

if [[ -z "${TIMEOUT_SECONDS:-}" ]]; then
    echo "TIMEOUT_SECONDS environment variable is not set. Using default value of 60 seconds."
    TIMEOUT_SECONDS=60
fi

echo "Installing Crossplane CLI with a timeout of $TIMEOUT_SECONDS seconds..."

curl -sL \
    "https://raw.githubusercontent.com/crossplane/crossplane/master/install.sh" \
    -o install-crossplane.sh
sudo chmod +x install-crossplane.sh

ending_time=$(($(date +%s) + TIMEOUT_SECONDS))

while true; do
    sh install-crossplane.sh || true
    if [[ -f crossplane ]]; then
        sudo mv crossplane /usr/local/bin/
        echo "Crossplane CLI installed successfully"
        break
    fi

    if [[ $(date +%s) -gt $ending_time ]]; then
        echo "Timeout while installing Crossplane CLI"
        exit 1
    fi

    sleep 5
done
