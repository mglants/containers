#!/usr/bin/env sh

set -e -u

RUNPATH="/usr/local/var/run/kea"

if [ -e "${RUNPATH}/kea-dhcp4.kea-dhcp4.pid" ]; then
    rm -f "${RUNPATH}/kea-dhcp4.kea-dhcp4.pid"
fi
if [ -e "${RUNPATH}/kea-dhcp6.kea-dhcp6.pid" ]; then
    rm -f "${RUNPATH}/kea-dhcp6.kea-dhcp6.pid"
fi
if [ -e "${RUNPATH}/kea-ctrl-agent.kea-ctrl-agent.pid" ]; then
        rm -f "${RUNPATH}/kea-ctrl-agent.kea-ctrl-agent.pid"
fi

keactrl start -c /etc/kea/keactrl.conf
sleep 10
set +e +u
if [[ -e /tmp/kea4-ctrl-socket ]] && [[ -e /tmp/kea6-ctrl-socket ]];then
  python3 /usr/local/bin/kea-exporter /tmp/kea4-ctrl-socket /tmp/kea6-ctrl-socket
elif [[ -e /tmp/kea4-ctrl-socket ]];then
  python3 /usr/local/bin/kea-exporter /tmp/kea4-ctrl-socket
elif [[ -e /tmp/kea6-ctrl-socket ]];then
  python3 /usr/local/bin/kea-exporter /tmp/kea6-ctrl-socket
fi
tail -f /dev/null