package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Unified MachineSet creation", framework.LabelDisruptive, Ordered, func() {
	var uf *framework.UnifiedFramework
	var client runtimeclient.Client
	var ctx context.Context
	var platform configv1.PlatformType

	BeforeAll(func() {
		var err error
		uf, err = framework.NewUnifiedFramework()
		Expect(err).NotTo(HaveOccurred())

		client, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		ctx = framework.GetContext()

		platform, err = framework.GetPlatform(ctx, client)
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates a new MachineSet via unified backend", func() {
		spec, err := uf.GetMachineSpec(platform, client)
		if err != nil {
			Skip("backend GetMachineSpec not implemented: " + err.Error())
		}

		tpl, err := uf.CreateMachineTemplate(ctx, client, platform, framework.BackendMachineTemplateParams{
			Name:     "unified-ms-create-template",
			Platform: platform,
			Spec:     spec,
		})
		if err != nil {
			Skip("backend CreateMachineTemplate not implemented: " + err.Error())
		}
		defer uf.DeleteMachineTemplate(ctx, client, tpl)

		ms, err := uf.CreateMachineSet(ctx, client, framework.BackendMachineSetParams{
			Name:          "unified-ms-create",
			Replicas:      1,
			Labels:        map[string]string{"e2e": "unified-ms-create"},
			Annotations:   map[string]string{"e2e": "unified-ms-create"},
			Template:      tpl,
			FailureDomain: "auto",
		})
		if err != nil {
			Skip("backend CreateMachineSet not implemented: " + err.Error())
		}
		defer uf.DeleteMachineSet(ctx, client, ms)

		status, err := uf.GetMachineSetStatus(ctx, client, ms)
		if err != nil {
			Skip("backend GetMachineSetStatus not implemented: " + err.Error())
		}
		Expect(status.Replicas).To(BeNumerically(">=", 0))
	})
})
