package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	yaml "sigs.k8s.io/yaml"
)

// mapiBackend implements the MachineBackend interface for MAPI backend
type mapiBackend struct {
	backendType      BackendType
	authoritativeAPI BackendType
}

func (m *mapiBackend) GetBackendType() BackendType      { return m.backendType }
func (m *mapiBackend) GetAuthoritativeAPI() BackendType { return m.authoritativeAPI }
func (m *mapiBackend) CreateMachineSet(ctx context.Context, client runtimeclient.Client, params BackendMachineSetParams) (interface{}, error) {
	machineSet := &machinev1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        params.Name,
			Namespace:   MachineAPINamespace,
			Labels:      params.Labels,
			Annotations: params.Annotations,
		},
		Spec: machinev1.MachineSetSpec{
			Replicas: &params.Replicas,
			Selector: metav1.LabelSelector{MatchLabels: map[string]string{"machine.openshift.io/cluster-api-machineset": params.Name}},
			Template: machinev1.MachineTemplateSpec{
				ObjectMeta: machinev1.ObjectMeta{Labels: MergeLabels(params.Labels, map[string]string{"machine.openshift.io/cluster-api-machineset": params.Name})},
				Spec:       machinev1.MachineSpec{},
			},
		},
	}

	// Set AuthoritativeAPI based on backend configuration
	if m.authoritativeAPI == BackendTypeCAPI {
		machineSet.Spec.Template.Spec.AuthoritativeAPI = machinev1.MachineAuthorityClusterAPI
	} else {
		machineSet.Spec.Template.Spec.AuthoritativeAPI = machinev1.MachineAuthorityMachineAPI
	}

	// Set ProviderSpec based on params.Template
	if params.Template != nil {
		providerSpec, err := m.convertTemplateToProviderSpec(ctx, client, params.Template)
		Expect(err).NotTo(HaveOccurred(), "Should convert template to provider spec")
		machineSet.Spec.Template.Spec.ProviderSpec = *providerSpec
	}

	Eventually(client.Create(ctx, machineSet), WaitMedium, RetryMedium).Should(Succeed(), "Should create MAPI MachineSet %s", machineSet.Name)
	return machineSet, nil
}

func (m *mapiBackend) DeleteMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	if ms, ok := machineSet.(*machinev1.MachineSet); ok {
		Eventually(func() error {
			return client.Delete(ctx, ms)
		}, time.Minute, RetryShort).Should(SatisfyAny(
			Succeed(),
			WithTransform(apierrors.IsNotFound, BeTrue()),
		), "Should delete MachineSet %s/%s successfully or MachineSet should not be found",
			ms.Namespace, ms.Name)
		return nil
	}
	return fmt.Errorf("invalid machine set type for MAPI backend")
}

func (m *mapiBackend) WaitForMachineSetDeleted(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	if ms, ok := machineSet.(*machinev1.MachineSet); ok {
		Eventually(komega.Get(ms), 10*time.Minute, 5*time.Second).Should(
			MatchError(ContainSubstring("not found")),
			"Should have MAPI MachineSet %s/%s deleted", ms.Namespace, ms.Name)
		return nil
	}
	return fmt.Errorf("invalid machine set type for MAPI backend")
}

func (m *mapiBackend) WaitForMachinesRunning(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	if ms, ok := machineSet.(*machinev1.MachineSet); ok {
		machineList := &machinev1.MachineList{}
		Eventually(func() error {
			if err := komega.List(machineList, runtimeclient.InNamespace(ms.Namespace), runtimeclient.MatchingLabels(ms.Spec.Selector.MatchLabels))(); err != nil {
				return err
			}

			for _, machine := range machineList.Items {
				if machine.Status.Phase == nil || *machine.Status.Phase != "Running" {
					var phase string
					if machine.Status.Phase != nil {
						phase = *machine.Status.Phase
					} else {
						phase = "nil"
					}
					return fmt.Errorf("machine %s is not in Running phase, current phase: %s", machine.Name, phase)
				}
			}

			if len(machineList.Items) == 0 {
				return fmt.Errorf("no machines found for MachineSet %s", ms.Name)
			}

			if ms.Spec.Replicas != nil && int32(len(machineList.Items)) != *ms.Spec.Replicas {
				return fmt.Errorf("expected %d machines, found %d", *ms.Spec.Replicas, len(machineList.Items))
			}

			return nil
		}, 10*time.Minute, 5*time.Second).Should(Succeed(),
			"Should have all machines in MAPI MachineSet %s/%s in Running phase", ms.Namespace, ms.Name)
		return nil
	}
	return fmt.Errorf("invalid machine set type for MAPI backend")
}

func (m *mapiBackend) GetMachineSetStatus(ctx context.Context, client runtimeclient.Client, machineSet interface{}) (*MachineSetStatus, error) {
	if ms, ok := machineSet.(*machinev1.MachineSet); ok {
		status := &MachineSetStatus{
			Replicas:          0,
			AvailableReplicas: ms.Status.AvailableReplicas,
			ReadyReplicas:     ms.Status.ReadyReplicas,
			AuthoritativeAPI:  "MachineAPI",
		}
		if ms.Spec.Replicas != nil {
			status.Replicas = *ms.Spec.Replicas
		}
		if ms.Status.AuthoritativeAPI != "" {
			status.AuthoritativeAPI = string(ms.Status.AuthoritativeAPI)
		}
		return status, nil
	}
	return nil, fmt.Errorf("invalid machine set type for MAPI backend")
}

func (m *mapiBackend) GetNodesFromMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) ([]corev1.Node, error) {
	if _, ok := machineSet.(*machinev1.MachineSet); ok {
		// TODO: Implement node query based on labels
		return []corev1.Node{}, nil
	}
	return nil, fmt.Errorf("invalid machine set type for MAPI backend")
}

func (m *mapiBackend) CreateMachineTemplate(ctx context.Context, client runtimeclient.Client, platform configv1.PlatformType, params BackendMachineTemplateParams) (interface{}, error) {
	switch platform {
	case configv1.AWSPlatformType:
		return m.createAWSMachineTemplate(ctx, client, params)
	case configv1.AzurePlatformType:
		return nil, fmt.Errorf("Azure machine template creation not yet implemented")
	case configv1.GCPPlatformType:
		return nil, fmt.Errorf("GCP machine template creation not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

func (m *mapiBackend) DeleteMachineTemplate(ctx context.Context, client runtimeclient.Client, template interface{}) error {
	return nil
}

func (m *mapiBackend) GetMachineSpec(platform configv1.PlatformType, client runtimeclient.Client) (interface{}, error) {
	switch platform {
	case configv1.AWSPlatformType:
		return m.getDefaultAWSMAPIProviderSpec(client)
	case configv1.AzurePlatformType:
		return nil, fmt.Errorf("Azure machine spec retrieval not yet implemented")
	case configv1.GCPPlatformType:
		return nil, fmt.Errorf("GCP machine spec retrieval not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// createAWSMachineTemplate creates AWS machine template, returns MAPI ProviderSpec
func (m *mapiBackend) createAWSMachineTemplate(ctx context.Context, client runtimeclient.Client, params BackendMachineTemplateParams) (interface{}, error) {
	// Get the default AWS MAPI ProviderSpec
	mapiProviderSpec, err := m.getDefaultAWSMAPIProviderSpec(client)
	Expect(err).NotTo(HaveOccurred(), "Should get default AWS MAPI provider spec")

	// For MAPI backend, we directly return serialized ProviderSpec instead of creating CAPI template
	// This is because MAPI backend uses ProviderSpec, not independent template resources

	// Serialize ProviderSpec to RawExtension
	providerSpecBytes, err := json.Marshal(mapiProviderSpec)
	Expect(err).NotTo(HaveOccurred(), "Should marshal provider spec")

	providerSpecRaw := &runtime.RawExtension{
		Raw: providerSpecBytes,
	}

	// Apply custom configuration from params.Spec if provided
	if params.Spec != nil {
		// Apply the configuration directly to the ProviderSpec
		configErr := ConfigureMachineTemplate(providerSpecRaw, params.Spec.(*MachineTemplateConfig))
		Expect(configErr).NotTo(HaveOccurred(), "Should apply custom configuration to MAPI ProviderSpec")
	}

	return providerSpecRaw, nil
}

// getDefaultAWSMAPIProviderSpec gets default AWS MAPI ProviderSpec
func (m *mapiBackend) getDefaultAWSMAPIProviderSpec(client runtimeclient.Client) (*machinev1.AWSMachineProviderConfig, error) {
	machineSetList := &machinev1.MachineSetList{}

	// List existing MAPI MachineSets
	Eventually(client.List(context.Background(), machineSetList, runtimeclient.InNamespace(MachineAPINamespace)), WaitMedium, RetryMedium).Should(Succeed(), "Should list MAPI machinesets in namespace %s", MachineAPINamespace)

	Expect(machineSetList.Items).NotTo(BeEmpty(), "Should have MAPI machinesets")

	// Use the first MachineSet's ProviderSpec as template
	machineSet := &machineSetList.Items[0]
	Expect(machineSet.Spec.Template.Spec.ProviderSpec.Value).NotTo(BeNil(), "Should have MAPI machineset ProviderSpec value")

	providerSpec := &machinev1.AWSMachineProviderConfig{}
	err := yaml.Unmarshal(machineSet.Spec.Template.Spec.ProviderSpec.Value.Raw, providerSpec)
	Expect(err).NotTo(HaveOccurred(), "Should unmarshal MAPI provider spec")

	return providerSpec, nil
}

// convertTemplateToProviderSpec converts CAPI template to MAPI ProviderSpec
func (m *mapiBackend) convertTemplateToProviderSpec(ctx context.Context, client runtimeclient.Client, template interface{}) (*machinev1.ProviderSpec, error) {
	switch t := template.(type) {
	case *awsv1.AWSMachineTemplate:
		return m.convertAWSTemplateToProviderSpec(ctx, client, t)
	case *runtime.RawExtension:
		// If already RawExtension, use directly
		return &machinev1.ProviderSpec{Value: t}, nil
	default:
		return nil, fmt.Errorf("unsupported template type: %T", template)
	}
}

// convertAWSTemplateToProviderSpec converts AWS CAPI template to MAPI ProviderSpec
func (m *mapiBackend) convertAWSTemplateToProviderSpec(ctx context.Context, client runtimeclient.Client, awsTemplate *awsv1.AWSMachineTemplate) (*machinev1.ProviderSpec, error) {
	// Get default AWS MAPI ProviderSpec as base
	defaultProviderSpec, err := m.getDefaultAWSMAPIProviderSpec(client)
	Expect(err).NotTo(HaveOccurred(), "Should get default AWS MAPI provider spec")

	// Update MAPI ProviderSpec using CAPI template configuration
	awsSpec := awsTemplate.Spec.Template.Spec

	// Update instance type
	if awsSpec.InstanceType != "" {
		defaultProviderSpec.InstanceType = awsSpec.InstanceType
	}

	// Update AMI information
	if awsSpec.AMI.ID != nil {
		defaultProviderSpec.AMI.ID = awsSpec.AMI.ID
	}

	// Update IAM instance profile
	if awsSpec.IAMInstanceProfile != "" {
		defaultProviderSpec.IAMInstanceProfile = &machinev1.AWSResourceReference{
			ID: &awsSpec.IAMInstanceProfile,
		}
	}

	// Update subnet configuration
	if awsSpec.Subnet != nil {
		if awsSpec.Subnet.ID != nil {
			defaultProviderSpec.Subnet = machinev1.AWSResourceReference{
				ID: awsSpec.Subnet.ID,
			}
		} else if len(awsSpec.Subnet.Filters) > 0 {
			filters := make([]machinev1.Filter, len(awsSpec.Subnet.Filters))
			for i, filter := range awsSpec.Subnet.Filters {
				filters[i] = machinev1.Filter{
					Name:   filter.Name,
					Values: filter.Values,
				}
			}
			defaultProviderSpec.Subnet = machinev1.AWSResourceReference{
				Filters: filters,
			}
		}
	}

	// Update security group configuration
	if len(awsSpec.AdditionalSecurityGroups) > 0 {
		securityGroups := make([]machinev1.AWSResourceReference, len(awsSpec.AdditionalSecurityGroups))
		for i, sg := range awsSpec.AdditionalSecurityGroups {
			if sg.ID != nil {
				securityGroups[i] = machinev1.AWSResourceReference{
					ID: sg.ID,
				}
			} else if len(sg.Filters) > 0 {
				filters := make([]machinev1.Filter, len(sg.Filters))
				for j, filter := range sg.Filters {
					filters[j] = machinev1.Filter{
						Name:   filter.Name,
						Values: filter.Values,
					}
				}
				securityGroups[i] = machinev1.AWSResourceReference{
					Filters: filters,
				}
			}
		}
		defaultProviderSpec.SecurityGroups = securityGroups
	}

	// Serialize to JSON
	providerSpecBytes, err := json.Marshal(defaultProviderSpec)
	Expect(err).NotTo(HaveOccurred(), "Should marshal provider spec")

	return &machinev1.ProviderSpec{
		Value: &runtime.RawExtension{
			Raw: providerSpecBytes,
		},
	}, nil
}
