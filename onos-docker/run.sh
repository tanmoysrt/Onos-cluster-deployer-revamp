# !/bin/sh


HOSTNAME=$(hostname)
export EXTRA_JAVA_OPTS="-Donos.cluster.metadata.uri=$METADATA_URL?hostname=$HOSTNAME"

# Start ONOS
/root/onos/bin/onos-service server
