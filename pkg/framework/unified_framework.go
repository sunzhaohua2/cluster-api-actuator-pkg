package framework

import (
	"context"

	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UnifiedFramework provides a unified testing framework interface
type UnifiedFramework struct {
	config  *TestConfig
	backend MachineBackend
}

// NewUnifiedFramework creates a new unified testing framework
func NewUnifiedFramework() *UnifiedFramework {
	config := LoadTestConfig()
	backend, err := config.GetBackend()
	Expect(err).NotTo(HaveOccurred(), "Should create backend")

	return &UnifiedFramework{
		config:  config,
		backend: backend,
	}
}

// GetBackendType returns the backend type
func (uf *UnifiedFramework) GetBackendType() BackendType {
	return uf.config.BackendType
}

// GetAuthoritativeAPI returns the authoritative API type
func (uf *UnifiedFramework) GetAuthoritativeAPI() BackendType {
	return uf.config.AuthoritativeAPI
}

// CreateMachineSet creates a machine set
func (uf *UnifiedFramework) CreateMachineSet(ctx context.Context, client runtimeclient.Client, params BackendMachineSetParams) (interface{}, error) {
	return uf.backend.CreateMachineSet(ctx, client, params)
}

// DeleteMachineSet deletes a machine set
func (uf *UnifiedFramework) DeleteMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	return uf.backend.DeleteMachineSet(ctx, client, machineSet)
}

// WaitForMachineSetDeleted waits for machine set deletion
func (uf *UnifiedFramework) WaitForMachineSetDeleted(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	return uf.backend.WaitForMachineSetDeleted(ctx, client, machineSet)
}

// WaitForMachinesRunning waits for all machines belonging to the machine set to enter the "Running" phase
func (uf *UnifiedFramework) WaitForMachinesRunning(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	return uf.backend.WaitForMachinesRunning(ctx, client, machineSet)
}

// GetMachineSetStatus returns the machine set status
func (uf *UnifiedFramework) GetMachineSetStatus(ctx context.Context, client runtimeclient.Client, machineSet interface{}) (*MachineSetStatus, error) {
	return uf.backend.GetMachineSetStatus(ctx, client, machineSet)
}

// GetNodesFromMachineSet returns nodes from a machine set
func (uf *UnifiedFramework) GetNodesFromMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) ([]corev1.Node, error) {
	return uf.backend.GetNodesFromMachineSet(ctx, client, machineSet)
}

// CreateMachineTemplate creates a machine template
func (uf *UnifiedFramework) CreateMachineTemplate(ctx context.Context, client runtimeclient.Client, platform configv1.PlatformType, params BackendMachineTemplateParams) (interface{}, error) {
	return uf.backend.CreateMachineTemplate(ctx, client, platform, params)
}

// DeleteMachineTemplate deletes a machine template
func (uf *UnifiedFramework) DeleteMachineTemplate(ctx context.Context, client runtimeclient.Client, template interface{}) error {
	return uf.backend.DeleteMachineTemplate(ctx, client, template)
}

// GetMachineSpec returns machine specification
func (uf *UnifiedFramework) GetMachineSpec(platform configv1.PlatformType, client runtimeclient.Client) (interface{}, error) {
	return uf.backend.GetMachineSpec(platform, client)
}

// GetTestDescription returns test description
func (uf *UnifiedFramework) GetTestDescription() string {
	return uf.config.GetTestDescription()
}

func (uf *UnifiedFramework) ReconfigureBackend(newBackendType, newAuthoritativeAPI BackendType) {
	newConfig := &TestConfig{
		BackendType:      newBackendType,
		AuthoritativeAPI: newAuthoritativeAPI,
	}

	newBackend, err := newConfig.GetBackend()
	Expect(err).NotTo(HaveOccurred(), "Should get backend")

	uf.config = newConfig
	uf.backend = newBackend
}
