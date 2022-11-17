#!/bin/sh

go run ./vendor/github.com/onsi/ginkgo/v2/ginkgo \
    -v \
    --timeout=115m \
    --grace-period=5m \
    --fail-fast \
    --no-color \
    --junit-report="junit_cluster_api_actuator_pkg_e2e.xml" \
    --output-dir="${OUTPUT_DIR}" \
    "$@" \
    ./pkg/ -- --alsologtostderr -v 4 -kubeconfig ${KUBECONFIG:-~/.kube/config}
