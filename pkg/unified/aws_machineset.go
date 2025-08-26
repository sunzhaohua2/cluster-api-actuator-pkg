package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("Unified MachineSet creation on aws", framework.LabelDisruptive, Ordered, func() {
	var uf *framework.UnifiedFramework
	var cl runtimeclient.Client
	var ctx context.Context
	var platform configv1.PlatformType
	var machineSpec interface{}
	var helper *TestHelper

	BeforeAll(func() {
		var err error
		uf = framework.NewUnifiedFramework()
		cl, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Should load client")
		komega.SetClient(cl)
		ctx = framework.GetContext()
		platform, err = framework.GetPlatform(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Should get platform")

		// Initialize test helper
		helper = NewTestHelper(ctx, uf, cl, platform, machineSpec)
		helper.SkipIfNotPlatform(configv1.AWSPlatformType)

		// Get machine spec once for all tests
		machineSpec, err = uf.GetMachineSpec(platform, cl)
		Expect(err).NotTo(HaveOccurred(), "Should get machine spec")
	})

	It("creates a new MachineSet via unified backend", func() {
		tpl := helper.CreateTemplate("unified-ms-template")
		defer helper.DeleteTemplate(tpl)
		ms := helper.CreateMachineSet("unified-ms", tpl, nil)
		defer helper.DeleteMachineSet(ms)

		// This will fail for mapi machineset with capi authority
		// TODO: wait for https://github.com/openshift/cluster-capi-operator/pull/365 merge
		uf.WaitForMachinesRunning(ctx, cl, ms)
	})

	It("creates a spot instance MachineSet via unified backend", func() {
		// Create spot configuration
		config := &framework.MachineTemplateConfig{
			AWS: &framework.AWSMachineConfig{
				SpotMarketOptions: &framework.SpotMarketConfig{
					MaxPrice: nil, // Use default price
				},
				Tenancy: func() *string { s := "default"; return &s }(),
			},
		}

		// Create template with spot configuration applied during creation
		tpl, err := uf.CreateMachineTemplate(ctx, cl, platform, framework.BackendMachineTemplateParams{
			Name:     "unified-spot-template",
			Platform: platform,
			Spec:     config, // Pass configuration during template creation
		})
		Expect(err).NotTo(HaveOccurred(), "Should create Machine Template with spot configuration")
		defer helper.DeleteTemplate(tpl)

		ms := helper.CreateMachineSet("unified-spot-ms", tpl, nil)
		defer helper.DeleteMachineSet(ms)
		// This will fail for mapi machineset with capi authority
		// TODO: wait for https://github.com/openshift/cluster-capi-operator/pull/365 merge
		uf.WaitForMachinesRunning(ctx, cl, ms)

		By("Verifying spot instance configuration contains spot-related fields")
		helper.VerifyMachineSetContainsString(ms, "spot")
	})
})
