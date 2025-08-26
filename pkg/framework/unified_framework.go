package framework

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UnifiedFramework 提供统一的测试框架接口
type UnifiedFramework struct {
	config  *TestConfig
	backend MachineBackend
}

// NewUnifiedFramework 创建新的统一测试框架
func NewUnifiedFramework() (*UnifiedFramework, error) {
	config := LoadTestConfig()
	backend, err := config.GetBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}
	
	return &UnifiedFramework{
		config:  config,
		backend: backend,
	}, nil
}

// GetBackendType 获取后端类型
func (uf *UnifiedFramework) GetBackendType() BackendType {
	return uf.config.BackendType
}

// GetAuthoritativeAPI 获取权威API类型
func (uf *UnifiedFramework) GetAuthoritativeAPI() BackendType {
	return uf.config.AuthoritativeAPI
}

// IsConversionLayerEnabled 检查是否启用转换层测试
func (uf *UnifiedFramework) IsConversionLayerEnabled() bool {
	return uf.config.IsConversionLayerEnabled()
}

// CreateMachineSet 创建机器集
func (uf *UnifiedFramework) CreateMachineSet(ctx context.Context, client runtimeclient.Client, params BackendMachineSetParams) (interface{}, error) {
	return uf.backend.CreateMachineSet(ctx, client, params)
}

// DeleteMachineSet 删除机器集
func (uf *UnifiedFramework) DeleteMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	return uf.backend.DeleteMachineSet(ctx, client, machineSet)
}

// WaitForMachineSetDeleted 等待机器集删除
func (uf *UnifiedFramework) WaitForMachineSetDeleted(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	return uf.backend.WaitForMachineSetDeleted(ctx, client, machineSet)
}

// GetMachineSetStatus 获取机器集状态
func (uf *UnifiedFramework) GetMachineSetStatus(ctx context.Context, client runtimeclient.Client, machineSet interface{}) (*MachineSetStatus, error) {
	return uf.backend.GetMachineSetStatus(ctx, client, machineSet)
}

// GetNodesFromMachineSet 从机器集获取节点
func (uf *UnifiedFramework) GetNodesFromMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) ([]corev1.Node, error) {
	return uf.backend.GetNodesFromMachineSet(ctx, client, machineSet)
}

// CreateMachineTemplate 创建机器模板
func (uf *UnifiedFramework) CreateMachineTemplate(ctx context.Context, client runtimeclient.Client, platform configv1.PlatformType, params BackendMachineTemplateParams) (interface{}, error) {
	return uf.backend.CreateMachineTemplate(ctx, client, platform, params)
}

// DeleteMachineTemplate 删除机器模板
func (uf *UnifiedFramework) DeleteMachineTemplate(ctx context.Context, client runtimeclient.Client, template interface{}) error {
	return uf.backend.DeleteMachineTemplate(ctx, client, template)
}

// GetMachineSpec 获取机器规格
func (uf *UnifiedFramework) GetMachineSpec(platform configv1.PlatformType, client runtimeclient.Client) (interface{}, error) {
	return uf.backend.GetMachineSpec(platform, client)
}

// RunMachineSetTest 运行机器集测试的通用方法
func (uf *UnifiedFramework) RunMachineSetTest(ctx context.Context, client runtimeclient.Client, testName string, testFunc func(context.Context, runtimeclient.Client, interface{}) error) error {
	// 根据平台类型获取机器规格
	platform, err := GetPlatform(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to get platform: %w", err)
	}
	
	machineSpec, err := uf.GetMachineSpec(platform, client)
	if err != nil {
		return fmt.Errorf("failed to get machine spec: %w", err)
	}
	
	// 创建机器模板
	templateParams := BackendMachineTemplateParams{
		Name:     fmt.Sprintf("%s-template", testName),
		Platform: platform,
		Spec:     machineSpec,
	}
	
	template, err := uf.CreateMachineTemplate(ctx, client, platform, templateParams)
	if err != nil {
		return fmt.Errorf("failed to create machine template: %w", err)
	}
	defer uf.DeleteMachineTemplate(ctx, client, template)
	
	// 创建机器集
	machineSetParams := BackendMachineSetParams{
		Name:         testName,
		Replicas:     1,
		Labels:       map[string]string{"test": testName},
		Annotations:  map[string]string{"test": testName},
		Template:     template,
		FailureDomain: "auto", // 这里需要根据平台动态设置
	}
	
	machineSet, err := uf.CreateMachineSet(ctx, client, machineSetParams)
	if err != nil {
		return fmt.Errorf("failed to create machine set: %w", err)
	}
	defer uf.DeleteMachineSet(ctx, client, machineSet)
	
	// 运行测试
	if err := testFunc(ctx, client, machineSet); err != nil {
		return fmt.Errorf("test failed: %w", err)
	}
	
	return nil
}

// GetTestDescription 获取测试描述
func (uf *UnifiedFramework) GetTestDescription() string {
	return uf.config.GetTestDescription()
}
