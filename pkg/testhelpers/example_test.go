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

package testhelpers_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/testhelpers"
)

// This file demonstrates how to use the minimal abstraction pattern
// for testing both MAPI and CAPI MachineSets.
//
// NOTE: Based on the team discussion, this pattern should be used for
// INFORMING tests only. For production CAPI GA tests, use CreateCAPIMachineSet directly.

var _ = Describe("MachineSet Helper Functions", Ordered, func() {
	var (
		namespace  string
		testConfig *testhelpers.TestConfig
		// k8sClient client.Client // Uncomment when running actual tests
	)

	BeforeAll(func() {
		// Get test configuration from environment
		testConfig = testhelpers.GetTestConfig()
		namespace = "default" // Or create a test namespace
	})

	Context("[Informing] Backend-agnostic tests", func() {
		// These tests use CreateMachineSetForBackend and work with both MAPI and CAPI
		// They are intended for informing tests to catch behavior differences

		It("Should create and delete MachineSet successfully", func(ctx SpecContext) {
			Skip("Example test - enable when k8sClient is available")

			By("Creating MachineSet using helper function")
			ms, err := testhelpers.CreateMachineSetForBackend(ctx, nil, testConfig.Backend, testhelpers.MachineSetParams{
				Name:      "test-machineset",
				Namespace: namespace,
				Replicas:  3,
				Labels: map[string]string{
					"test": "example",
				},
				// For CAPI, platform-specific fields are required
				Platform:     "AWS",
				InstanceType: "m5.large",
				ClusterName:  "test-cluster",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(ms).NotTo(BeNil())

			By("Verifying MachineSet was created")
			Expect(ms.GetName()).To(Equal("test-machineset"))
			Expect(ms.GetNamespace()).To(Equal(namespace))

			// Migration note (2030):
			// When MAPI is removed, this test code requires ZERO changes
			// because it uses the helper function abstraction.
		})

		It("Should scale MachineSet up and down", func(ctx SpecContext) {
			Skip("Example test - enable when k8sClient is available")

			By("Creating MachineSet with 1 replica")
			ms, err := testhelpers.CreateMachineSetForBackend(ctx, nil, testConfig.Backend, testhelpers.MachineSetParams{
				Name:         "test-scale",
				Namespace:    namespace,
				Replicas:     1,
				Platform:     "AWS",
				InstanceType: "m5.large",
				ClusterName:  "test-cluster",
			})
			Expect(err).NotTo(HaveOccurred())

			By("Scaling up to 3 replicas")
			err = testhelpers.ScaleMachineSet(ctx, nil, ms, 3)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for MachineSet to be ready")
			err = testhelpers.WaitForMachineSetReady(ctx, nil, ms, 3, 5*time.Minute)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying status")
			status, err := testhelpers.GetMachineSetStatus(ctx, nil, ms)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.ReadyReplicas).To(Equal(int32(3)))

			By("Cleaning up")
			err = testhelpers.DeleteMachineSet(ctx, nil, ms)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Migration scenario", func() {
		// Tests specific to the migration period (MAPI backend + CAPI authority)

		It("Should handle non-authoritative MAPI MachineSets", func(ctx SpecContext) {
			if !testConfig.IsMigrationScenario() {
				Skip("Only runs in migration scenario (TEST_BACKEND=mapi TEST_AUTHORITY=capi)")
			}

			Skip("Example test - implement migration-specific logic")

			By("Creating MAPI MachineSet (synced from CAPI)")
			ms, err := testhelpers.CreateMachineSetForBackend(ctx, nil, testConfig.Backend, testhelpers.MachineSetParams{
				Name:         "non-auth-ms",
				Namespace:    namespace,
				Replicas:     2,
				Platform:     "AWS",
				InstanceType: "m5.large",
				ClusterName:  "test-cluster",
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying MachineSet is marked as non-authoritative")
			// Add validation logic here
			// e.g., check that spec.authoritativeAPI is set to CAPI

			By("Cleaning up")
			err = testhelpers.DeleteMachineSet(ctx, nil, ms)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("AWS Spot Instance support", func() {
		// Tests for AWS Spot Instance feature

		It("Should create Spot Instance MachineSet", func(ctx SpecContext) {
			Skip("Example test - enable when k8sClient is available")

			By("Creating MachineSet with Spot instances")
			maxPrice := "" // Empty string means on-demand price
			ms, err := testhelpers.CreateMachineSetForBackend(ctx, nil, testConfig.Backend, testhelpers.MachineSetParams{
				Name:         "test-spot",
				Namespace:    namespace,
				Replicas:     2,
				Platform:     "AWS",
				InstanceType: "m5.large",
				ClusterName:  "test-cluster",
				SpotMaxPrice: &maxPrice, // Enable Spot instances
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying MachineSet was created")
			Expect(ms.GetName()).To(Equal("test-spot"))

			if testConfig.Backend == testhelpers.BackendCAPI {
				By("Verifying AWSMachineTemplate has Spot configuration")
				// For CAPI, check that the template has SpotMarketOptions
				template := &capav1.AWSMachineTemplate{}
				err = client.IgnoreNotFound(nil)
				_ = template
				// err = k8sClient.Get(ctx, client.ObjectKey{
				// 	Name:      "test-spot-template",
				// 	Namespace: namespace,
				// }, template)
				// Expect(err).NotTo(HaveOccurred())
				// Expect(template.Spec.Template.Spec.SpotMarketOptions).NotTo(BeNil())
				// Expect(template.Spec.Template.Spec.SpotMarketOptions.MaxPrice).To(Equal(&maxPrice))
			}

			By("Cleaning up")
			err = testhelpers.DeleteMachineSet(ctx, nil, ms)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create Spot Instance with custom max price", func(ctx SpecContext) {
			Skip("Example test - enable when k8sClient is available")

			By("Creating MachineSet with custom Spot price")
			maxPrice := "0.50" // $0.50/hour maximum
			ms, err := testhelpers.CreateMachineSetForBackend(ctx, nil, testConfig.Backend, testhelpers.MachineSetParams{
				Name:         "test-spot-custom",
				Namespace:    namespace,
				Replicas:     1,
				Platform:     "AWS",
				InstanceType: "m5.large",
				ClusterName:  "test-cluster",
				SpotMaxPrice: &maxPrice,
			})
			Expect(err).NotTo(HaveOccurred())

			if testConfig.Backend == testhelpers.BackendCAPI {
				By("Verifying custom max price")
				// Verification logic here
			}

			By("Cleaning up")
			err = testhelpers.DeleteMachineSet(ctx, nil, ms)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Backend-specific tests", func() {
		// These tests demonstrate how to write backend-specific logic
		// when absolutely necessary

		It("Should test MAPI-specific features", func(ctx SpecContext) {
			if testConfig.Backend != testhelpers.BackendMAPI {
				Skip("MAPI-only test")
				// This entire test will be DELETED in 2030
			}

			Skip("Implement MAPI-specific test logic")
			// Test MAPI-specific features here
		})

		It("Should test CAPI-specific features", func(ctx SpecContext) {
			if testConfig.Backend != testhelpers.BackendCAPI {
				Skip("CAPI-only test")
				// This test SURVIVES after 2030
			}

			Skip("Implement CAPI-specific test logic")
			// Test CAPI-specific features here
		})
	})
})

// Example of direct CAPI test (recommended for GA)
var _ = Describe("CAPI MachineSet Direct Tests", func() {
	// These tests use CreateCAPIMachineSet directly
	// Recommended for production CAPI GA tests

	It("Should create CAPI MachineSet directly", func(ctx SpecContext) {
		Skip("Example test - enable when k8sClient is available")

		By("Creating CAPI MachineSet using direct function")
		ms, err := testhelpers.CreateCAPIMachineSet(ctx, nil, testhelpers.MachineSetParams{
			Name:         "capi-direct",
			Namespace:    "default",
			Replicas:     3,
			Platform:     "AWS",
			InstanceType: "m5.large",
			ClusterName:  "test-cluster",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(ms).NotTo(BeNil())

		// This approach is recommended for CAPI GA tests
		// because it's explicit, simple, and doesn't require backend abstraction
	})

	It("Should create CAPI Spot Instance directly", func(ctx SpecContext) {
		Skip("Example test - enable when k8sClient is available")

		maxPrice := ""
		ms, err := testhelpers.CreateCAPIMachineSet(ctx, nil, testhelpers.MachineSetParams{
			Name:         "capi-spot-direct",
			Namespace:    "default",
			Replicas:     2,
			Platform:     "AWS",
			InstanceType: "m5.large",
			ClusterName:  "test-cluster",
			SpotMaxPrice: &maxPrice,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(ms).NotTo(BeNil())
	})
})
