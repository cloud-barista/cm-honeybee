#!/bin/bash
# Uninstall cm-honeybee-agent from this host (inverse of copyAgent.sh).
# Removes the binary, systemd unit, config and install leftovers.

is_root() {
    [[ "$EUID" -ne 0 ]] && return 1 || return 0
}

root_check() {
    if ! is_root; then
        echo "Please run as root!"
        exit 1
    fi
}

Uninstall() {
    systemctl stop cm-honeybee-agent > /dev/null 2>&1
    systemctl disable cm-honeybee-agent > /dev/null 2>&1
    rm -f /lib/systemd/system/cm-honeybee-agent.service
    systemctl daemon-reload > /dev/null 2>&1

    rm -f /usr/bin/cm-honeybee-agent
    rm -rf /etc/cloud-migrator/cm-honeybee-agent

    # Install-time leftovers.
    rm -f /tmp/honeybee-agent-install.log /tmp/busybox /tmp/busybox-arm64

    echo "cm-honeybee-agent uninstalled."
}

# Main Script
root_check
Uninstall
