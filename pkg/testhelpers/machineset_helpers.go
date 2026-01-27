/*
Copyright 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testhelpers

import (
	"context"
	"fmt"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	capibuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/cluster-api/core/v1beta2"
	infrabuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/cluster-api/infrastructure/v1beta2"
	mapibuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/machine/v1beta1"
)

// BackendType represents the API backend type.
type BackendType string

const (
	// BackendMAPI represents the Machine API backend.
	BackendMAPI BackendType = "mapi"
	// BackendCAPI represents the Cluster API backend.
	BackendCAPI BackendType = "capi"
)

// MachineSetParams contains common parameters for creating MachineSets.
type MachineSetParams struct {
	Name         string
	Namespace    string
	Replicas     int32
	Labels       map[string]string
	Annotations  map[string]string
	ClusterName  string
	GenerateName string

	// Platform-specific fields (for CAPI)
	Platform     configv1.PlatformType // AWS, Azure, GCP, etc.
	InstanceType string                // e.g., "m5.large" for AWS
	AMI          string                // For AWS
	IAMProfile   string                // For AWS
	Subnet       *capav1.AWSResourceReference

	// Spot instance configuration (for AWS)
	SpotMaxPrice *string // Maximum price for spot instances (empty string = on-demand price)
}

// MachineSetStatus contains common status fields from MachineSets.
type MachineSetStatus struct {
	Replicas          int32
	ReadyReplicas     int32
	AvailableReplicas int32
}

// CreateMAPIMachineSet creates a MAPI MachineSet with the given parameters.
// This function will be REMOVED when MAPI is deprecated (~2030).
func CreateMAPIMachineSet(ctx context.Context, cl client.Client, params MachineSetParams) (*machinev1.MachineSet, error) {
	builder := mapibuilder.MachineSet().
		WithName(params.Name).
		WithNamespace(params.Namespace).
		WithReplicas(params.Replicas)

	if params.GenerateName != "" {
		builder = builder.WithGenerateName(params.GenerateName)
	}

	if params.Labels != nil {
		builder = builder.WithLabels(params.Labels)
	}

	if params.Annotations != nil {
		builder = builder.WithAnnotations(params.Annotations)
	}

	ms := builder.Build()

	if err := cl.Create(ctx, ms); err != nil {
		return nil, fmt.Errorf("failed to create MAPI MachineSet: %w", err)
	}

	return ms, nil
}

// CreateCAPIMachineSet creates a CAPI MachineSet with the given parameters.
// For CAPI, this requires creating a platform-specific MachineTemplate first.
func CreateCAPIMachineSet(ctx context.Context, cl client.Client, params MachineSetParams) (*clusterv1.MachineSet, error) {
	// Step 1: Create platform-specific MachineTemplate
	var infraRef clusterv1.ContractVersionedObjectReference

	switch params.Platform {
	case configv1.AWSPlatformType, "":
		// Default to AWS if not specified
		template, err := createAWSMachineTemplate(ctx, cl, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create AWS MachineTemplate: %w", err)
		}
		infraRef = clusterv1.ContractVersionedObjectReference{
			APIGroup: "infrastructure.cluster.x-k8s.io",
			Kind:     "AWSMachineTemplate",
			Name:     template.Name,
		}

	// TODO: Add support for other platforms (Azure, GCP, etc.)
	// case configv1.AzurePlatformType:
	//     template, err := createAzureMachineTemplate(ctx, cl, params)
	//     ...
	// case configv1.GCPPlatformType:
	//     template, err := createGCPMachineTemplate(ctx, cl, params)
	//     ...

	default:
		return nil, fmt.Errorf("unsupported platform for CAPI: %s", params.Platform)
	}

	// Step 2: Create MachineSet with reference to the template
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"machineset": params.Name,
		},
	}

	templateLabels := map[string]string{
		"machineset": params.Name,
	}
	if params.Labels != nil {
		for k, v := range params.Labels {
			templateLabels[k] = v
		}
	}

	builder := capibuilder.MachineSet().
		WithName(params.Name).
		WithNamespace(params.Namespace).
		WithReplicas(params.Replicas).
		WithSelector(selector).
		WithTemplate(clusterv1.MachineTemplateSpec{
			ObjectMeta: clusterv1.ObjectMeta{
				Labels: templateLabels,
			},
			Spec: clusterv1.MachineSpec{
				ClusterName: params.ClusterName,
				Bootstrap: clusterv1.Bootstrap{
					DataSecretName: nil, // TODO: Add bootstrap data if needed
				},
				InfrastructureRef: infraRef,
			},
		})

	if params.ClusterName != "" {
		builder = builder.WithClusterName(params.ClusterName)
	}

	if params.GenerateName != "" {
		builder = builder.WithGenerateName(params.GenerateName)
	}

	if params.Labels != nil {
		builder = builder.WithLabels(params.Labels)
	}

	if params.Annotations != nil {
		builder = builder.WithAnnotations(params.Annotations)
	}

	ms := builder.Build()

	if err := cl.Create(ctx, ms); err != nil {
		return nil, fmt.Errorf("failed to create CAPI MachineSet: %w", err)
	}

	return ms, nil
}

// createAWSMachineTemplate creates an AWSMachineTemplate for CAPI MachineSet.
func createAWSMachineTemplate(ctx context.Context, cl client.Client, params MachineSetParams) (*capav1.AWSMachineTemplate, error) {
	templateName := params.Name + "-template"
	if params.GenerateName != "" {
		templateName = params.GenerateName + "template-"
	}

	builder := infrabuilder.AWSMachineTemplate().
		WithName(templateName).
		WithNamespace(params.Namespace)

	if params.InstanceType != "" {
		builder = builder.WithInstanceType(params.InstanceType)
	} else {
		// Default instance type
		builder = builder.WithInstanceType("m5.large")
	}

	if params.AMI != "" {
		builder = builder.WithAMI(capav1.AMIReference{
			ID: &params.AMI,
		})
	}

	if params.IAMProfile != "" {
		builder = builder.WithIAMInstanceProfile(params.IAMProfile)
	}

	if params.Subnet != nil {
		builder = builder.WithSubnet(params.Subnet)
	}

	// Configure Spot instances if requested
	if params.SpotMaxPrice != nil {
		builder = builder.WithSpotMarketOptions(&capav1.SpotMarketOptions{
			MaxPrice: params.SpotMaxPrice,
		})
	}

	template := builder.Build()

	if err := cl.Create(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to create AWSMachineTemplate: %w", err)
	}

	return template, nil
}

// CreateMachineSetForBackend creates a MachineSet for the specified backend type.
//
// NOTE: This function is intended for INFORMING tests only.
// For production CAPI tests, use CreateCAPIMachineSet directly.
//
// Migration path (2030):
//  1. Delete CreateMAPIMachineSet function
//  2. Remove BackendMAPI case from this switch
//  3. Simplify this function to directly call CreateCAPIMachineSet
//  4. Test code using this function requires minimal changes
func CreateMachineSetForBackend(ctx context.Context, cl client.Client, backend BackendType, params MachineSetParams) (client.Object, error) {
	switch backend {
	case BackendMAPI:
		return CreateMAPIMachineSet(ctx, cl, params)
	case BackendCAPI:
		return CreateCAPIMachineSet(ctx, cl, params)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", backend)
	}
}

// GetMachineSetStatus retrieves common status fields from a MachineSet.
//
// Migration path (2030):
//  1. Remove MAPI type assertion
//  2. Simplify to only handle CAPI MachineSet
func GetMachineSetStatus(ctx context.Context, cl client.Client, ms client.Object) (*MachineSetStatus, error) {
	key := client.ObjectKeyFromObject(ms)

	switch obj := ms.(type) {
	case *machinev1.MachineSet:
		// MAPI MachineSet
		updated := &machinev1.MachineSet{}
		if err := cl.Get(ctx, key, updated); err != nil {
			return nil, fmt.Errorf("failed to get MAPI MachineSet: %w", err)
		}
		return &MachineSetStatus{
			Replicas:          updated.Status.Replicas,
			ReadyReplicas:     updated.Status.ReadyReplicas,
			AvailableReplicas: updated.Status.AvailableReplicas,
		}, nil

	case *clusterv1.MachineSet:
		// CAPI MachineSet
		updated := &clusterv1.MachineSet{}
		if err := cl.Get(ctx, key, updated); err != nil {
			return nil, fmt.Errorf("failed to get CAPI MachineSet: %w", err)
		}

		replicas := int32(0)
		readyReplicas := int32(0)
		availableReplicas := int32(0)

		if updated.Status.Replicas != nil {
			replicas = *updated.Status.Replicas
		}
		if updated.Status.ReadyReplicas != nil {
			readyReplicas = *updated.Status.ReadyReplicas
		}
		if updated.Status.AvailableReplicas != nil {
			availableReplicas = *updated.Status.AvailableReplicas
		}

		return &MachineSetStatus{
			Replicas:          replicas,
			ReadyReplicas:     readyReplicas,
			AvailableReplicas: availableReplicas,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported MachineSet type: %T", obj)
	}
}

// DeleteMachineSet deletes a MachineSet regardless of backend type.
//
// Migration path (2030):
//  1. No changes needed - client.Delete works for any client.Object
func DeleteMachineSet(ctx context.Context, cl client.Client, ms client.Object) error {
	if err := cl.Delete(ctx, ms); err != nil {
		return fmt.Errorf("failed to delete MachineSet: %w", err)
	}
	return nil
}

// WaitForMachineSetReady waits for a MachineSet to reach the desired replica count.
//
// Migration path (2030):
//  1. No changes needed - uses GetMachineSetStatus which will be simplified
func WaitForMachineSetReady(ctx context.Context, cl client.Client, ms client.Object, desiredReplicas int32, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		status, err := GetMachineSetStatus(ctx, cl, ms)
		if err != nil {
			return fmt.Errorf("failed to get MachineSet status: %w", err)
		}

		if status.ReadyReplicas == desiredReplicas && status.AvailableReplicas == desiredReplicas {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// Continue polling
		}
	}

	return fmt.Errorf("timeout waiting for MachineSet to be ready")
}

// ScaleMachineSet scales a MachineSet to the desired replica count.
//
// Migration path (2030):
//  1. Remove MAPI type assertion
//  2. Simplify to only handle CAPI MachineSet
func ScaleMachineSet(ctx context.Context, cl client.Client, ms client.Object, replicas int32) error {
	switch obj := ms.(type) {
	case *machinev1.MachineSet:
		obj.Spec.Replicas = &replicas
		if err := cl.Update(ctx, obj); err != nil {
			return fmt.Errorf("failed to scale MAPI MachineSet: %w", err)
		}
		return nil

	case *clusterv1.MachineSet:
		obj.Spec.Replicas = &replicas
		if err := cl.Update(ctx, obj); err != nil {
			return fmt.Errorf("failed to scale CAPI MachineSet: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported MachineSet type: %T", obj)
	}
}
