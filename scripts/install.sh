#!/bin/bash
source scripts/helper.sh

NAMESPACE=$(__readini global NAMESPACE)
INFLUXDB_HOST=$(__readini influxdb INFLUXDB_HOST)
INFLUXDB_PV_PATH=$(__readini influxdb INFLUXDB_PV_PATH)
GRAFANA_HOST=$(__readini grafana GRAFANA_HOST)
GRAFANA_PV_PATH=$(__readini grafana GRAFANA_PV_PATH)
export NAMESPACE
export INFLUXDB_PV_PATH
export INFLUXDB_HOST
export GRAFANA_HOST
export GRAFANA_PV_PATH

# install influxdb
function __install_fluxdb {
    echo "install influxdb..."
    INFLUX_ADDR=http://influxdb.$NAMESPACE.svc.cluster.local:8086
    INFLUX_USERNAME=admin
    INFLUX_ORG=kubebpfbox
    INFLUX_PASSWORD=$(__generate_password)
    INFLUX_BUCKET=kubebpfbox
    cat deploy/influxdb/pv.yaml | envsubst | kubectl apply -f -
    cat deploy/influxdb/statefulset.yaml | envsubst | kubectl apply -f -
    __wait_sts $NAMESPACE influxdb
    # init influxdb user info
    kubectl exec -it  influxdb-0 -n $NAMESPACE -- influx setup \
        --username $INFLUX_USERNAME \
        --password $INFLUX_PASSWORD \
        --bucket default \
        --org $INFLUX_ORG \
        --force
    
    # create default bucket
    bucketid=$(kubectl exec -it  influxdb-0 -n $NAMESPACE -- influx bucket create \
        --org=$INFLUX_ORG \
        --name=$INFLUX_BUCKET \
        --retention 2d \
        | tail -n 1 | awk '{print $1}')
    # create token
    INFLUX_TOKEN=$(kubectl exec -it  influxdb-0 -n $NAMESPACE -- influx auth create \
        --org=$INFLUX_ORG \
        --read-bucket $bucketid \
        --write-bucket $bucketid \
        --description "for-kubebpfbox" \
        | tail -n 1  | awk '{print $3}')

    echo INFLUX_ADDR:     $INFLUX_ADDR  > .influxdb_info
    echo INFLUX_USERNAME: $INFLUX_USERNAME  >> .influxdb_info
    echo INFLUX_PASSWORD: $INFLUX_PASSWORD  >> .influxdb_info
    echo INFLUX_ORG:      $INFLUX_ORG  >> .influxdb_info
    echo INFLUX_BUCKET:   $INFLUX_BUCKET  >> .influxdb_info
    echo INFLUX_TOKEN:    $INFLUX_TOKEN  >> .influxdb_info
    export INFLUX_ADDR
    export INFLUX_ORG
    export INFLUX_BUCKET
    export INFLUX_TOKEN
}

# install grafana
function __install_grafana {
    echo "install grafana..."
    cat deploy/grafana/configmap-config.yaml | envsubst | kubectl apply -f -
    cat deploy/grafana/configmap-provisioning-dashboards.yaml | envsubst | kubectl apply -f -
    cat deploy/grafana/configmap-datasource.yaml  | envsubst | kubectl apply -f -
    cat deploy/grafana/pv.yaml  | envsubst | kubectl apply -f -
    cat deploy/grafana/deployment.yaml | envsubst | kubectl apply -f -
    __wait_deploy $NAMESPACE grafana
}

# install kubebpfbox agent
function __install_agent {
    echo "install agent..."
    cat deploy/agent/role.yaml | envsubst | kubectl apply -f -
    cat deploy/agent/daemonset.yaml | envsubst | kubectl apply -f -
}
# install
__install_fluxdb
__install_grafana
__install_agent

echo "install end..."
