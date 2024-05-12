#!/bin/sh

# Get the initial list of UIDs
UIDS=$(kubectl get pods -l app.kubernetes.io/instance=$STATEFULSET_LABEL -o jsonpath='{range .items[*]}{.metadata.uid}{"\n"}{end}' | sort)

# Generate the initial configuration files
generate_config() {
  # Get the index of the pod in the stateful set
  

  # Get the list of pod IPs and generate the peer entries
  PEER_ENTRIES=""
  while IFS=$'\t' read -r UID NAME IP; do
    INDEX=$(echo "$NAME" | awk -F '-' '{print $NF}')
    PEER_ENTRIES="$PEER_ENTRIES{\"name\":\"$NAME\",\"url\":\"http://$IP:8000/\",\"role\":\"$(if [ $INDEX -eq 0 ]; then echo "primary"; else echo "secondary"; fi)\",\"auto-failover\":true},"
  done < <(kubectl get pods -l app.kubernetes.io/instance=$STATEFULSET_LABEL -o jsonpath='{range .items[*]}{.metadata.uid}{"\t"}{.metadata.name}{"\t"}{.status.podIP}{"\n"}{end}' | sort -k1,1)

  # Remove the trailing comma from the last peer entry
  PEER_ENTRIES="${PEER_ENTRIES%,}"

  # Generate the JSON object with the list of peers
  echo "\"peers\": [$PEER_ENTRIES]" > /etc/kea/peers.json
  # Reload kea-configuration
  curl -X POST -H "Content-Type: application/json" -d '{ "command": "config-reload", "service": [ "dhcp4","dhcp6" ] }' http://127.0.0.1:8000/
}
echo "First initialization"
cp /etc/kea/ha-init.json /etc/kea/ha.json
generate_config
curl -X POST -H "Content-Type: application/json" -d '{ "command": "libreload", "service": [ "dhcp4","dhcp6" ] }' http://127.0.0.1:8000/
curl -X POST -H "Content-Type: application/json" -d '{ "command": "config-reload", "service": [ "dhcp4","dhcp6" ] }' http://127.0.0.1:8000/
# Watch for changes to the UIDs and regenerate the configuration files when necessary
while true; do
  # Get the current list of UIDs
  NEW_UIDS=$(kubectl get pods -l app.kubernetes.io/instance=$STATEFULSET_LABEL -o jsonpath='{range .items[*]}{.metadata.uid}{"\n"}{end}' | sort)

  # Check if the UIDs have changed
  if [ "$UIDS" != "$NEW_UIDS" ]; then
    echo "PODS Restarted reload ip addresses in kea"
    # Update the UIDs and regenerate the configuration files
    UIDS=$NEW_UIDS
    generate_config
  fi

  # Wait for 10 seconds before checking again
  sleep 10
done
