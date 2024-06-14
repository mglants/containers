#!/bin/bash
mkdir -p /data/{inbox,outbox,sent,error}
envsubst < /scripts/gammu.conf.pre \
         > ~/app/gammu.conf
# Run the original command
exec "$@"
