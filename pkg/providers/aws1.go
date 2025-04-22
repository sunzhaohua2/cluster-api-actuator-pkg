package providers

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	mapiv1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
	capiinfrastructurev1beta2resourcebuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/cluster-api/infrastructure/v1beta2"
	awsprovider "github.com/openshift/cluster-api-provider-aws/pkg/apis/awsprovider/v1beta1"
	corev1 "k8s.io/api/core/v1"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml"
)

const (
	awsMachineTemplateName = "aws-machine-template"
	infrastructureName     = "cluster"
	infraAPIVersion        = "infrastructure.cluster.x-k8s.io/v1beta1"
)

type AWSMachineSet struct {
	Client               client.Client
	MachineSet           *clusterv1.MachineSet
	MachineSpec          *mapiv1.AWSMachineProviderConfig
	AWSMachineTemplate   *awsprovider.AWSMachineTemplate
	ClusterName          string
	NamePrefix           string
}

func (a *AWSMachineSet) CreateMachineSet(ctx context.Context) error {
	// 构造 MachineTemplate
	a.AWSMachineTemplate = newAWSMachineTemplate(a.Client, a.MachineSpec)
	if err := a.Client.Create(ctx, a.AWSMachineTemplate); err != nil {
		return fmt.Errorf("failed to create AWSMachineTemplate: %w", err)
	}

	// 构造 MachineSet
	var err error
	a.MachineSet, err = framework.CreateCAPIMachineSet(ctx, a.Client, framework.NewCAPIMachineSetParams(
		a.NamePrefix,
		a.ClusterName,
		a.MachineSpec.Placement.AvailabilityZone,
		1,
		corev1.ObjectReference{
			Kind:       "AWSMachineTemplate",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Name:       a.AWSMachineTemplate.Name,
		}))
	if err != nil {
		return fmt.Errorf("failed to create AWS MachineSet: %w", err)
	}

	framework.WaitForCAPIMachinesRunning(ctx, a.Client, a.MachineSet.Name)
	return nil
}

func (a *AWSMachineSet) Cleanup(ctx context.Context) {
	if a.MachineSet != nil {
		framework.DeleteCAPIMachineSets(ctx, a.Client, a.MachineSet)
		framework.WaitForCAPIMachineSetsDeleted(ctx, a.Client, a.MachineSet)
	}
	if a.AWSMachineTemplate != nil {
		framework.DeleteObjects(ctx, a.Client, a.AWSMachineTemplate)
	}
}

var _ = Describe("Cluster API AWS MachineSet", framework.LabelCAPI, framework.LabelDisruptive, Ordered, func() {
	var (
		cl                      client.Client
		ctx                     = context.Background()
		platform                configv1.PlatformType
		clusterName             string
		oc                      *gatherer.CLI
		awsMachineTemplate      *awsv1.AWSMachineTemplate
		machineSetParams        framework.CAPIMachineSetParams
		machineSet              *clusterv1.MachineSet
		mapiDefaultProviderSpec *mapiv1.AWSMachineProviderConfig
		err                     error
	)

	BeforeAll(func() {
		cfg, err := config.GetConfig()
		Expect(err).ToNot(HaveOccurred(), "Failed to GetConfig")

		cl, err = client.New(cfg, client.Options{})
		Expect(err).ToNot(HaveOccurred(), "Failed to create Kubernetes client for test")

		infra := &configv1.Infrastructure{}
		infraName := client.ObjectKey{
			Name: infrastructureName,
		}
		Expect(cl.Get(ctx, infraName, infra)).To(Succeed(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.PlatformStatus).ToNot(BeNil(), "expected the infrastructure Status.PlatformStatus to not be nil")
		clusterName = infra.Status.InfrastructureName
		platform = infra.Status.PlatformStatus.Type
		if platform != configv1.AWSPlatformType {
			Skip("Skipping AWS E2E tests")
		}
		oc, err = framework.NewCLI()
		Expect(err).ToNot(HaveOccurred(), "Failed to new CLI")
		framework.SkipIfNotTechPreviewNoUpgrade(oc, cl)
		_, mapiDefaultProviderSpec = getDefaultAWSMAPIProviderSpec(cl)
		machineSetParams = framework.NewCAPIMachineSetParams(
			"aws-machineset",
			clusterName,
			mapiDefaultProviderSpec.Placement.AvailabilityZone,
			1,
			corev1.ObjectReference{
				Kind:       "AWSMachineTemplate",
				APIVersion: infraAPIVersion,
				Name:       awsMachineTemplateName,
			},
		)
		framework.CreateCoreCluster(ctx, cl, clusterName, "AWSCluster")
	})

	AfterEach(func() {
		framework.DeleteCAPIMachineSets(ctx, cl, machineSet)
		framework.WaitForCAPIMachineSetsDeleted(ctx, cl, machineSet)
		framework.DeleteObjects(ctx, cl, awsMachineTemplate)
	})

	It("should be able to run a machine with a default provider spec", func() {
		awsMachineTemplate = newAWSMachineTemplate(mapiDefaultProviderSpec)
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-51071", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})
	
})

func getDefaultAWSMAPIProviderSpec(cl client.Client) (*mapiv1.MachineSet, *mapiv1.AWSMachineProviderConfig) {
	machineSetList := &mapiv1.MachineSetList{}

	Eventually(func() error {
		return cl.List(framework.GetContext(), machineSetList, client.InNamespace(framework.MachineAPINamespace))
	}, framework.WaitShort, framework.RetryShort).Should(Succeed(), "it should be able to list the MAPI machinesets")
	Expect(machineSetList.Items).ToNot(HaveLen(0), "expected the MAPI machinesets to be present")

	machineSet := &machineSetList.Items[0]
	Expect(machineSet.Spec.Template.Spec.ProviderSpec.Value).ToNot(BeNil(), "expected the MAPI machinesets ProviderSpec value to not be nil")

	providerSpec := &mapiv1.AWSMachineProviderConfig{}
	Expect(yaml.Unmarshal(machineSet.Spec.Template.Spec.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "it should be able to unmarshal the raw yaml into providerSpec")

	return machineSet, providerSpec
}

func newAWSMachineTemplate(mapiProviderSpec *mapiv1.AWSMachineProviderConfig) *awsv1.AWSMachineTemplate {
	By("Creating AWS machine template")

	Expect(mapiProviderSpec).ToNot(BeNil(), "expected the mapi ProviderSpec to not be nil")
	Expect(mapiProviderSpec.IAMInstanceProfile).ToNot(BeNil(), "expected the mapi IAMInstanceProfile to not be nil")
	Expect(mapiProviderSpec.IAMInstanceProfile.ID).ToNot(BeNil(), "expected the mapi IAMInstanceProfile.ID to not be nil")
	Expect(mapiProviderSpec.InstanceType).ToNot(BeEmpty(), "expected the mapi InstanceType to not be empty")
	Expect(mapiProviderSpec.Placement.AvailabilityZone).ToNot(BeEmpty(), "expected the mapi Placement.AvailabilityZone to not be empty")
	Expect(mapiProviderSpec.AMI.ID).ToNot(BeNil(), "expected the mapi AMI.ID to not be nil")
	Expect(mapiProviderSpec.SecurityGroups).ToNot(HaveLen(0), "expected the mapi SecurityGroups to be present")
	Expect(mapiProviderSpec.SecurityGroups[0].Filters).ToNot(HaveLen(0), "expected the mapi SecurityGroups[0].Filters to be present")
	Expect(mapiProviderSpec.SecurityGroups[0].Filters[0].Values).ToNot(HaveLen(0), "expected the mapi SecurityGroups[0].Filters[0].Values to be present")

	var subnet awsv1.AWSResourceReference

	if len(mapiProviderSpec.Subnet.Filters) == 0 {
		subnet = awsv1.AWSResourceReference{
			ID: mapiProviderSpec.Subnet.ID,
		}
	} else {
		subnet = awsv1.AWSResourceReference{
			Filters: []awsv1.Filter{
				{
					Name:   "tag:Name",
					Values: mapiProviderSpec.Subnet.Filters[0].Values,
				},
			},
		}
	}

	uncompressedUserData := true
	ami := awsv1.AMIReference{
		ID: mapiProviderSpec.AMI.ID,
	}
	ignition := &awsv1.Ignition{
		Version:     "3.4",
		StorageType: awsv1.IgnitionStorageTypeOptionUnencryptedUserData,
	}
	additionalSecurityGroups := []awsv1.AWSResourceReference{
		{
			Filters: []awsv1.Filter{
				{
					Name:   "tag:Name",
					Values: mapiProviderSpec.SecurityGroups[0].Filters[0].Values,
				},
			},
		},
		{
			Filters: []awsv1.Filter{
				{
					Name:   "tag:Name",
					Values: mapiProviderSpec.SecurityGroups[1].Filters[0].Values,
				},
			},
		},
	}
	awsmt := capiinfrastructurev1beta2resourcebuilder.
		AWSMachineTemplate().
		WithUncompressedUserData(uncompressedUserData).
		WithIAMInstanceProfile(*mapiProviderSpec.IAMInstanceProfile.ID).
		WithInstanceType(mapiProviderSpec.InstanceType).
		WithAMI(ami).
		WithIgnition(ignition).
		WithSubnet(&subnet).
		WithAdditionalSecurityGroups(additionalSecurityGroups).
		WithName(awsMachineTemplateName).
		WithNamespace(framework.ClusterAPINamespace).
		Build()

	return awsmt
}
