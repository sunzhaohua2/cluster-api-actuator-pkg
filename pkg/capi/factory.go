package capi

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewMachineSetAPI(platform string, client client.Client, clusterName string) (MachineSetAPI, error) {
	switch platform {
	case "aws":
		machineSpec := getAWSMAPIProviderSpec(client) 
		return &AWSMachineSet{
			Client:      client,
			MachineSpec: machineSpec,
			ClusterName: clusterName,
			NamePrefix:  "aws-machineset",
		}, nil
	case "azure":
		machineSpec := getAzureMAPIProviderSpec(client)
		return &AzureMachineSet{
			Client:      client,
			MachineSpec: machineSpec,
			ClusterName: clusterName,
			NamePrefix:  "azure-machineset",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}
