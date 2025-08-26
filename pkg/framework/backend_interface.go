package framework

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// BackendType 定义后端类型
type BackendType string

const (
	BackendTypeMAPI BackendType = "MAPI"
	BackendTypeCAPI BackendType = "CAPI"
)

// MachineBackend 定义机器后端的抽象接口
type MachineBackend interface {
	GetBackendType() BackendType
	GetAuthoritativeAPI() BackendType

	CreateMachineSet(ctx context.Context, client runtimeclient.Client, params BackendMachineSetParams) (interface{}, error)
	DeleteMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error
	WaitForMachineSetDeleted(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error
	GetMachineSetStatus(ctx context.Context, client runtimeclient.Client, machineSet interface{}) (*MachineSetStatus, error)
	GetNodesFromMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) ([]corev1.Node, error)

	CreateMachineTemplate(ctx context.Context, client runtimeclient.Client, platform configv1.PlatformType, params BackendMachineTemplateParams) (interface{}, error)
	DeleteMachineTemplate(ctx context.Context, client runtimeclient.Client, template interface{}) error
	GetMachineSpec(platform configv1.PlatformType, client runtimeclient.Client) (interface{}, error)
}

// BackendMachineSetParams 定义创建机器集的通用参数
type BackendMachineSetParams struct {
	Name          string
	Replicas      int32
	Labels        map[string]string
	Annotations   map[string]string
	Template      interface{}
	FailureDomain string
}

// BackendMachineTemplateParams 定义创建机器模板的通用参数
type BackendMachineTemplateParams struct {
	Name     string
	Platform configv1.PlatformType
	Spec     interface{}
}

// MachineSetStatus 定义机器集状态的通用结构
type MachineSetStatus struct {
	Replicas          int32
	AvailableReplicas int32
	ReadyReplicas     int32
	Phase             string
}

// BackendFactory 创建后端实例的工厂
type BackendFactory struct{}

// NewBackend 根据配置创建相应的后端实例
func (f *BackendFactory) NewBackend(backendType BackendType, authoritativeAPI BackendType) (MachineBackend, error) {
	switch backendType {
	case BackendTypeMAPI:
		return NewMAPIBackend(authoritativeAPI), nil
	case BackendTypeCAPI:
		return NewCAPIBackend(authoritativeAPI), nil
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
}

func NewMAPIBackend(authoritativeAPI BackendType) MachineBackend {
	return &mapiBackend{backendType: BackendTypeMAPI, authoritativeAPI: authoritativeAPI}
}

func NewCAPIBackend(authoritativeAPI BackendType) MachineBackend {
	return &capiBackend{backendType: BackendTypeCAPI, authoritativeAPI: authoritativeAPI}
}
