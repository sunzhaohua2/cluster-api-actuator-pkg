package capi

import (
	"fmt"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/machinesetapi"
)

func NewCreator(platform string) (machinesetapi.Creator, error) {
    switch platform {
    case "aws":
        return &awsCreator{}, nil
    case "azure":
        return &azureCreator{}, nil
    case "gcp":
        return &gcpCreator{}, nil
    default:
        return nil, fmt.Errorf("unsupported platform: %s", platform)
    }
}