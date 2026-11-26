#!/bin/sh
set -eu

if [ -e /dev/kvm ]; then
    kvm_gid="$(stat -c '%g' /dev/kvm)"
    if [ "$kvm_gid" != "0" ]; then
        kvm_group="$(awk -F: -v gid="$kvm_gid" '$3 == gid { print $1; exit }' /etc/group)"
        if [ -z "$kvm_group" ]; then
            kvm_group="hostkvm"
            addgroup -g "$kvm_gid" -S "$kvm_group"
        fi
        if ! id -nG runner | tr ' ' '\n' | grep -qx "$kvm_group"; then
            addgroup runner "$kvm_group"
        fi
    fi
fi

exec su-exec runner "$@"
