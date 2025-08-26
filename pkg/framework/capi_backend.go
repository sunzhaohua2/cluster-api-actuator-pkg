package framework

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	azurev1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	gcpv1 "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

const (
	// DefaultAMIID is a placeholder AMI ID for testing
	// In production, this should be obtained from cluster configuration
	DefaultAMIID = "ami-0c55b159cbfafe1d0"
)

// capiBackend implements the MachineBackend interface for CAPI backend
type capiBackend struct {
	backendType      BackendType
	authoritativeAPI BackendType
}

func (c *capiBackend) GetBackendType() BackendType      { return c.backendType }
func (c *capiBackend) GetAuthoritativeAPI() BackendType { return c.authoritativeAPI }

func (c *capiBackend) CreateMachineSet(ctx context.Context, client runtimeclient.Client, params BackendMachineSetParams) (interface{}, error) {
	// Get cluster name
	infra, err := GetInfrastructure(ctx, client)
	Expect(err).NotTo(HaveOccurred(), "Should get infrastructure global object")
	Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "Should have infrastructure name on Infrastructure.Status")

	clusterName := infra.Status.InfrastructureName
	userDataSecret := "worker-user-data"
	machineSet := &clusterv1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        params.Name,
			Namespace:   ClusterAPINamespace,
			Labels:      params.Labels,
			Annotations: params.Annotations,
		},
		Spec: clusterv1.MachineSetSpec{
			ClusterName: clusterName,
			Replicas:    &params.Replicas,
			Selector:    metav1.LabelSelector{MatchLabels: map[string]string{"cluster.x-k8s.io/set-name": params.Name}},

			Template: clusterv1.MachineTemplateSpec{
				ObjectMeta: clusterv1.ObjectMeta{Labels: MergeLabels(params.Labels, map[string]string{"cluster.x-k8s.io/set-name": params.Name})},
				Spec: clusterv1.MachineSpec{
					Bootstrap: clusterv1.Bootstrap{
						DataSecretName: &userDataSecret,
					},
					ClusterName: clusterName,
				},
			},
		},
	}

	// Set InfrastructureRef based on params.Template
	if params.Template != nil {
		err := c.setInfrastructureRef(ctx, client, &machineSet.Spec.Template.Spec, params.Template)
		Expect(err).NotTo(HaveOccurred(), "Should set infrastructure ref")
	}

	Eventually(client.Create(ctx, machineSet), WaitMedium, RetryMedium).Should(Succeed(), "Should create CAPI MachineSet %s", machineSet.Name)
	return machineSet, nil
}

func (c *capiBackend) DeleteMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	if ms, ok := machineSet.(*clusterv1.MachineSet); ok {
		Eventually(func() error {
			return client.Delete(ctx, ms)
		}, time.Minute, RetryShort).Should(SatisfyAny(
			Succeed(),
			WithTransform(apierrors.IsNotFound, BeTrue()),
		), "Should delete MachineSet %s/%s successfully or MachineSet should not be found",
			ms.Namespace, ms.Name)
		return nil
	}
	return fmt.Errorf("invalid machine set type for CAPI backend")
}

func (c *capiBackend) WaitForMachineSetDeleted(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	if ms, ok := machineSet.(*clusterv1.MachineSet); ok {
		Eventually(komega.Get(ms), 10*time.Minute, 5*time.Second).Should(
			MatchError(ContainSubstring("not found")),
			"Should have CAPI MachineSet %s/%s deleted", ms.Namespace, ms.Name)
		return nil
	}
	return fmt.Errorf("invalid machine set type for CAPI backend")
}

func (c *capiBackend) WaitForMachinesRunning(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	if ms, ok := machineSet.(*clusterv1.MachineSet); ok {
		machineList := &clusterv1.MachineList{}
		Eventually(func() error {
			if err := komega.List(machineList, runtimeclient.InNamespace(ms.Namespace), runtimeclient.MatchingLabels(ms.Spec.Selector.MatchLabels))(); err != nil {
				return err
			}

			for _, machine := range machineList.Items {
				if machine.Status.Phase != string(clusterv1.MachinePhaseRunning) {
					return fmt.Errorf("machine %s is not in Running phase, current phase: %s", machine.Name, machine.Status.Phase)
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
			"Should have all machines in CAPI MachineSet %s/%s in Running phase", ms.Namespace, ms.Name)
		return nil
	}
	return fmt.Errorf("invalid machine set type for CAPI backend")
}

func (c *capiBackend) GetMachineSetStatus(ctx context.Context, client runtimeclient.Client, machineSet interface{}) (*MachineSetStatus, error) {
	if ms, ok := machineSet.(*clusterv1.MachineSet); ok {
		status := &MachineSetStatus{
			Replicas:          0,
			AvailableReplicas: ms.Status.AvailableReplicas,
			ReadyReplicas:     ms.Status.ReadyReplicas,
		}
		if ms.Spec.Replicas != nil {
			status.Replicas = *ms.Spec.Replicas
		}
		return status, nil
	}
	return nil, fmt.Errorf("invalid machine set type for CAPI backend")
}

func (c *capiBackend) GetNodesFromMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) ([]corev1.Node, error) {
	if _, ok := machineSet.(*clusterv1.MachineSet); ok {
		// TODO: Implement node query based on labels
		return []corev1.Node{}, nil
	}
	return nil, fmt.Errorf("invalid machine set type for CAPI backend")
}

func (c *capiBackend) CreateMachineTemplate(ctx context.Context, client runtimeclient.Client, platform configv1.PlatformType, params BackendMachineTemplateParams) (interface{}, error) {
	switch platform {
	case configv1.AWSPlatformType:
		return c.createAWSMachineTemplate(ctx, client, params)
	case configv1.AzurePlatformType:
		return nil, fmt.Errorf("Azure machine template creation not yet implemented")
	case configv1.GCPPlatformType:
		return nil, fmt.Errorf("GCP machine template creation not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

func (c *capiBackend) DeleteMachineTemplate(ctx context.Context, client runtimeclient.Client, template interface{}) error {
	switch tmpl := template.(type) {
	case *awsv1.AWSMachineTemplate:
		Eventually(func() error {
			return client.Delete(ctx, tmpl)
		}, time.Minute, RetryShort).Should(SatisfyAny(
			Succeed(),
			WithTransform(apierrors.IsNotFound, BeTrue()),
		), "Should delete AWS MachineTemplate %s/%s successfully or MachineTemplate should not be found",
			tmpl.Namespace, tmpl.Name)
		return nil
	case *azurev1.AzureMachineTemplate:
		Eventually(func() error {
			return client.Delete(ctx, tmpl)
		}, time.Minute, RetryShort).Should(SatisfyAny(
			Succeed(),
			WithTransform(apierrors.IsNotFound, BeTrue()),
		), "Should delete Azure MachineTemplate %s/%s successfully or MachineTemplate should not be found",
			tmpl.Namespace, tmpl.Name)
		return nil
	case *gcpv1.GCPMachineTemplate:
		Eventually(func() error {
			return client.Delete(ctx, tmpl)
		}, time.Minute, RetryShort).Should(SatisfyAny(
			Succeed(),
			WithTransform(apierrors.IsNotFound, BeTrue()),
		), "Should delete GCP MachineTemplate %s/%s successfully or MachineTemplate should not be found",
			tmpl.Namespace, tmpl.Name)
		return nil
	default:
		return fmt.Errorf("unsupported machine template type: %T", template)
	}
}

func (c *capiBackend) GetMachineSpec(platform configv1.PlatformType, client runtimeclient.Client) (interface{}, error) {
	switch platform {
	case configv1.AWSPlatformType:
		return c.getDefaultAWSCAPIMachineSpec(client)
	case configv1.AzurePlatformType:
		return nil, fmt.Errorf("Azure machine spec retrieval not yet implemented")
	case configv1.GCPPlatformType:
		return nil, fmt.Errorf("GCP machine spec retrieval not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// createAWSMachineTemplate creates AWS machine template
func (c *capiBackend) createAWSMachineTemplate(ctx context.Context, client runtimeclient.Client, params BackendMachineTemplateParams) (interface{}, error) {
	// Get default AWS CAPI machine specification
	awsMachineSpec, err := c.getDefaultAWSCAPIMachineSpec(client)
	Expect(err).NotTo(HaveOccurred(), "Should get default AWS CAPI machine spec")

	// Create AWS machine template
	awsMachineTemplate := &awsv1.AWSMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: ClusterAPINamespace,
		},
		Spec: awsv1.AWSMachineTemplateSpec{
			Template: awsv1.AWSMachineTemplateResource{
				Spec: *awsMachineSpec,
			},
		},
	}

	// Apply custom configuration from params.Spec if provided
	if params.Spec != nil {
		// Apply the configuration directly to the template before creating it
		configErr := ConfigureMachineTemplate(awsMachineTemplate, params.Spec.(*MachineTemplateConfig))
		Expect(configErr).NotTo(HaveOccurred(), "Should apply custom configuration to template before creation")
	}

	// Create template
	Eventually(client.Create(ctx, awsMachineTemplate), WaitMedium, RetryMedium).Should(Succeed(), "Should create AWS machine template %s", awsMachineTemplate.Name)

	return awsMachineTemplate, nil
}

// getDefaultAWSCAPIMachineSpec gets default AWS CAPI machine specification
func (c *capiBackend) getDefaultAWSCAPIMachineSpec(client runtimeclient.Client) (*awsv1.AWSMachineSpec, error) {
	// Find existing AWS machine templates
	awsTemplateList := &awsv1.AWSMachineTemplateList{}
	Eventually(client.List(context.Background(), awsTemplateList, runtimeclient.InNamespace(ClusterAPINamespace)), WaitMedium, RetryMedium).Should(Succeed(), "Should list AWS machine templates in namespace %s", ClusterAPINamespace)

	if len(awsTemplateList.Items) == 0 {
		// If no existing templates found, create a default one
		return c.createDefaultAWSCAPIMachineSpec(), nil
	}

	// Use the first template's specification as default
	return &awsTemplateList.Items[0].Spec.Template.Spec, nil
}

// createDefaultAWSCAPIMachineSpec creates default AWS CAPI machine specification
func (c *capiBackend) createDefaultAWSCAPIMachineSpec() *awsv1.AWSMachineSpec {
	return &awsv1.AWSMachineSpec{
		InstanceType: "m5.large",
		AMI: awsv1.AMIReference{
			// Use default AMI ID, should be obtained from cluster configuration in actual use
			ID: ptr.To(DefaultAMIID), // example AMI ID
		},
		Ignition: &awsv1.Ignition{
			Version:     "3.4",
			StorageType: awsv1.IgnitionStorageTypeOptionUnencryptedUserData,
		},
		Subnet: &awsv1.AWSResourceReference{
			// Use default subnet, should be obtained from cluster configuration in actual use
			Filters: []awsv1.Filter{
				{
					Name:   "tag:Name",
					Values: []string{"*worker*"},
				},
			},
		},
		AdditionalSecurityGroups: []awsv1.AWSResourceReference{
			{
				Filters: []awsv1.Filter{
					{
						Name:   "tag:Name",
						Values: []string{"*worker*"},
					},
				},
			},
		},
	}
}

// setInfrastructureRef sets InfrastructureRef based on template type
func (c *capiBackend) setInfrastructureRef(ctx context.Context, client runtimeclient.Client, machineSpec *clusterv1.MachineSpec, template interface{}) error {
	switch t := template.(type) {
	case *awsv1.AWSMachineTemplate:
		machineSpec.InfrastructureRef = corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
			Kind:       "AWSMachineTemplate",
			Name:       t.Name,
			Namespace:  t.Namespace,
		}
	case *azurev1.AzureMachineTemplate:
		machineSpec.InfrastructureRef = corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "AzureMachineTemplate",
			Name:       t.Name,
			Namespace:  t.Namespace,
		}
	case *gcpv1.GCPMachineTemplate:
		machineSpec.InfrastructureRef = corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "GCPMachineTemplate",
			Name:       t.Name,
			Namespace:  t.Namespace,
		}
	default:
		return fmt.Errorf("unsupported template type for infrastructure ref: %T", template)
	}
	return nil
}
