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

package testutils_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/testutils"
)

// This file demonstrates how to use the minimal abstraction pattern
// for testing both MAPI and CAPI MachineSets.
//
// Key advantages over full interface abstraction:
// 1. Simpler code - just helper functions, no complex interface
// 2. Easy migration - in 2030, just delete MAPI helper functions
// 3. Test code changes are minimal when removing MAPI

var _ = Describe("MachineSet Operations (Minimal Abstraction Pattern)", func() {
	var (
		testConfig *testutils.TestConfig
		namespace  string
	)

	BeforeEach(func() {
		// Get test configuration from environment variables
		testConfig = testutils.GetTestConfig()
		namespace = "test-namespace"

		// Log which scenario we're testing
		if testConfig.IsPureMAPI() {
			GinkgoLogr.Info("Testing pure MAPI scenario")
		} else if testConfig.IsPureCAPI() {
			GinkgoLogr.Info("Testing pure CAPI scenario")
		} else if testConfig.IsMigrationScenario() {
			GinkgoLogr.Info("Testing migration scenario (MAPI backend + CAPI authority)")
		}
	})

	Context("Basic MachineSet operations", func() {
		It("Should create and delete MachineSet successfully", func(ctx SpecContext) {
			By("Creating MachineSet using helper function")
			ms, err := testutils.CreateMachineSetForBackend(ctx, k8sClient, testConfig.Backend, testutils.MachineSetParams{
				Name:      "test-machineset",
				Namespace: namespace,
				Replicas:  3,
				Labels: map[string]string{
					"test": "example",
				},
				// For CAPI, platform-specific fields are required
				Platform:     "AWS", // or configv1.AWSPlatformType
				InstanceType: "m5.large",
				ClusterName:  "test-cluster",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(ms).NotTo(BeNil())

			By("Verifying MachineSet was created")
			Expect(ms.GetName()).To(Equal("test-machineset"))
			Expect(ms.GetNamespace()).To(Equal(namespace))

			By("Deleting MachineSet")
			err = testutils.DeleteMachineSet(ctx, k8sClient, ms)
			Expect(err).NotTo(HaveOccurred())

			// Migration note (2030):
			// When MAPI is removed, this test code requires ZERO changes
			// because it uses the helper function abstraction.
		})

		It("Should scale MachineSet up and down", func(ctx SpecContext) {
			By("Creating MachineSet with 1 replica")
			ms, err := testutils.CreateMachineSetForBackend(ctx, k8sClient, testConfig.Backend, testutils.MachineSetParams{
				Name:         "test-scale",
				Namespace:    namespace,
				Replicas:     1,
				Platform:     "AWS",
				InstanceType: "m5.large",
				ClusterName:  "test-cluster",
			})
			Expect(err).NotTo(HaveOccurred())

			By("Scaling up to 3 replicas")
			err = testutils.ScaleMachineSet(ctx, k8sClient, ms, 3)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for MachineSet to be ready")
			err = testutils.WaitForMachineSetReady(ctx, k8sClient, ms, 3, 5*time.Minute)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying status")
			status, err := testutils.GetMachineSetStatus(ctx, k8sClient, ms)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.ReadyReplicas).To(Equal(int32(3)))
			Expect(status.AvailableReplicas).To(Equal(int32(3)))

			By("Scaling down to 1 replica")
			err = testutils.ScaleMachineSet(ctx, k8sClient, ms, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Cleaning up")
			err = testutils.DeleteMachineSet(ctx, k8sClient, ms)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Migration scenario testing", func() {
		// This context demonstrates testing the MAPI ↔ CAPI migration scenario
		// Only relevant during 2025-2030 migration period

		It("Should handle non-authoritative MAPI MachineSets", func(ctx SpecContext) {
			if !testConfig.IsMigrationScenario() {
				Skip("Only runs in migration scenario (TEST_BACKEND=mapi TEST_AUTHORITY=capi)")
			}

			By("Creating MAPI MachineSet (synced from CAPI)")
			ms, err := testutils.CreateMachineSetForBackend(ctx, k8sClient, testConfig.Backend, testutils.MachineSetParams{
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
			err = testutils.DeleteMachineSet(ctx, k8sClient, ms)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create Spot Instance MachineSet", func(ctx SpecContext) {
			By("Creating MachineSet with Spot instances")
			maxPrice := "" // Empty string means on-demand price
			ms, err := testutils.CreateMachineSetForBackend(ctx, k8sClient, testConfig.Backend, testutils.MachineSetParams{
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

			if testConfig.Backend == testutils.BackendCAPI {
				By("Verifying AWSMachineTemplate has Spot configuration")
				// For CAPI, check that the template has SpotMarketOptions
				template := &capav1.AWSMachineTemplate{}
				err = k8sClient.Get(ctx, client.ObjectKey{
					Name:      "test-spot-template",
					Namespace: namespace,
				}, template)
				Expect(err).NotTo(HaveOccurred())
				Expect(template.Spec.Template.Spec.SpotMarketOptions).NotTo(BeNil())
				Expect(template.Spec.Template.Spec.SpotMarketOptions.MaxPrice).To(Equal(&maxPrice))
			}

			By("Cleaning up")
			err = testutils.DeleteMachineSet(ctx, k8sClient, ms)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create Spot Instance with custom max price", func(ctx SpecContext) {
			By("Creating MachineSet with custom Spot price")
			maxPrice := "0.50" // $0.50/hour maximum
			ms, err := testutils.CreateMachineSetForBackend(ctx, k8sClient, testConfig.Backend, testutils.MachineSetParams{
				Name:         "test-spot-custom",
				Namespace:    namespace,
				Replicas:     1,
				Platform:     "AWS",
				InstanceType: "m5.large",
				ClusterName:  "test-cluster",
				SpotMaxPrice: &maxPrice,
			})
			Expect(err).NotTo(HaveOccurred())

			if testConfig.Backend == testutils.BackendCAPI {
				By("Verifying custom max price")
				template := &capav1.AWSMachineTemplate{}
				err = k8sClient.Get(ctx, client.ObjectKey{
					Name:      "test-spot-custom-template",
					Namespace: namespace,
				}, template)
				Expect(err).NotTo(HaveOccurred())
				Expect(template.Spec.Template.Spec.SpotMarketOptions).NotTo(BeNil())
				Expect(*template.Spec.Template.Spec.SpotMarketOptions.MaxPrice).To(Equal("0.50"))
			}

			By("Cleaning up")
			err = testutils.DeleteMachineSet(ctx, k8sClient, ms)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Backend-specific tests", func() {
		// These tests demonstrate how to write backend-specific logic
		// when absolutely necessary

		It("Should test MAPI-specific features", func(ctx SpecContext) {
			if testConfig.Backend != testutils.BackendMAPI {
				Skip("MAPI-only test")
			}

			// MAPI-specific test logic
			// This entire test will be deleted in 2030
		})

		It("Should test CAPI-specific features", func(ctx SpecContext) {
			if testConfig.Backend != testutils.BackendCAPI {
				Skip("CAPI-only test")
			}

			// CAPI-specific test logic
			// This test survives after MAPI removal
		})
	})
})

// Example: How to run tests in different scenarios
//
// 1. Pure MAPI (traditional):
//    $ TEST_BACKEND=mapi TEST_AUTHORITY=mapi make test-e2e
//
// 2. Pure CAPI (future):
//    $ TEST_BACKEND=capi TEST_AUTHORITY=capi make test-e2e
//
// 3. Migration scenario (MAPI backend + CAPI authority):
//    $ TEST_BACKEND=mapi TEST_AUTHORITY=capi make test-e2e
//
// 4. Run all scenarios in CI:
//    $ make test-all-scenarios

// Example: 2030 Migration Process
//
// Step 1: Delete MAPI helper functions from machineset_helpers.go
//   - Remove CreateMAPIMachineSet()
//   - Remove MAPI case from CreateMachineSetForBackend()
//   - Remove MAPI type assertion from GetMachineSetStatus()
//   - Remove MAPI type assertion from ScaleMachineSet()
//
// Step 2: Simplify test_config.go
//   - Remove TEST_BACKEND environment variable handling
//   - Always return BackendCAPI
//   - Remove migration scenario functions
//
// Step 3: Clean up tests
//   - Delete MAPI-specific test contexts (marked with Skip)
//   - Delete migration scenario tests
//   - Keep all tests using CreateMachineSetForBackend() - they work unchanged!
//
// Step 4: Rename for clarity (optional)
//   - Rename CreateMachineSetForBackend() -> CreateMachineSet()
//   - Remove "ForBackend" suffix since there's only one backend
//
// Estimated effort: 2-3 days (vs 2-3 weeks for full interface removal)

var k8sClient client.Client // Mock client for example
