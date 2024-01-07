#!/bin/bash

# build and push agent image
docker build -t xygao/kubebpfbox-agent:0.1.0 -f build/agent/Dockerfile .
docker push xygao/kubebpfbox-agent:0.1.0

# build and push grafana image
docker build -t xygao/kubebpfbox-grafana:9.5.12 -f build/grafana/Dockerfile .
docker push xygao/kubebpfbox-grafana:9.5.12