package capi

import (
	"context"
	"errors"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/machineset"
	"k8s.io/apimachinery/pkg/runtime"
)

type AzureSpec struct {
	InstanceType       string `json:"instanceType"`
	AvailabilityZone   string `json:"availabilityZone"`
}

func (s *AzureSpec) Validate() error {
	if s.InstanceType == "" {
		return errors.New("instanceType is required")
	}
	return nil
}

func (s *AzureSpec) ToRawExtension() (runtime.RawExtension, error) {
	return runtime.DefaultUnstructuredConverter.ToUnstructured(s)
}

type azureCreator struct{}

func (a *azureCreator) Create(ctx context.Context, params machineset.CreateParams) (*machineset.MachineSet, error) {
	spec, ok := params.ProviderSpec.(*AzureSpec)
	if !ok {
		return nil, errors.New("invalid provider spec for Azure")
	}
	
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	
	rawSpec, err := spec.ToRawExtension()
	if err != nil {
		return nil, err
	}
	
	return &machineset.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: params.Name,
		},
		Replicas:    &params.Replicas,
		ProviderSpec: rawSpec,
	}, nil
}