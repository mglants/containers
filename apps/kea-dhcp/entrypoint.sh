#!/usr/bin/env sh

set -e -u

RUNPATH="/run/kea"

if [ -e "${RUNPATH}/kea-dhcp4.kea-dhcp4.pid" ]; then
    rm -f "${RUNPATH}/kea-dhcp4.kea-dhcp4.pid"
fi
if [ -e "${RUNPATH}/kea-dhcp6.kea-dhcp6.pid" ]; then
    rm -f "${RUNPATH}/kea-dhcp6.kea-dhcp6.pid"
fi
if [ -e "${RUNPATH}/kea-ctrl-agent.kea-ctrl-agent.pid" ]; then
        rm -f "${RUNPATH}/kea-ctrl-agent.kea-ctrl-agent.pid"
fi

keactrl start -c /config/kea/keactrl.conf
sleep 10
set +e +u
if [[ -e /run/kea/kea-dhcp4-ctrl.sock ]] && [[ -e /run/kea/kea-dhcp6-ctrl.sock ]];then
  python3 /usr/local/bin/kea-exporter /run/kea/kea-dhcp4-ctrl.sock /run/kea/kea-dhcp6-ctrl.sock
elif [[ -e /run/kea/kea-dhcp4-ctrl.sock ]];then
  python3 /usr/local/bin/kea-exporter /run/kea/kea-dhcp4-ctrl.sock
elif [[ -e /run/kea/kea-dhcp6-ctrl.sock ]];then
  python3 /usr/local/bin/kea-exporter /run/kea/kea-dhcp6-ctrl.sock
fi
tail -f /dev/null