package machinesetapi

import (
	"context"
	"fmt"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/machinesetapi/capi"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/machinesetapi/mapi"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Creator interface {
    Create(ctx context.Context, params CreateParams) (*MachineSet, error)
    Delete(ctx context.Context, name string) error
}

type CreateParams struct {
	Name         string
	Replicas     int32
	Platform     string // aws/azure/gcp
	ProviderSpec ProviderSpec
}

// factoary
func NewCreator(apiType string, platform string, client runtimeclient.Client) (Creator, error) {
	switch apiType {
	case "mapi":
		return mapi.NewCreator(client), nil
	case "capi":
		return capi.NewCreator(platform, client)
	default:
		return nil, fmt.Errorf("unsupported API type: %s", apiType)
	}
}