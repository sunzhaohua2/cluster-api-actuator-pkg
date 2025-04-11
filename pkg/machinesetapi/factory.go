package capi

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewMachineSetAPI(authType string, client client.Client, platform string, params MachineSetParams) (MachineSetAPI, error) {
    switch authType {
    case "mapi":
        return NewMAPI(client, platform, params)
    case "capi":
        return NewCAPI(client, platform, params)
    default:
        return nil, fmt.Errorf("unsupported API type: %s", authType)
    }
}
