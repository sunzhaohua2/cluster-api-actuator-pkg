package machinesetapi

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MachineSet 是 MAPI 和 CAPI 的通用抽象
type MachineSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	// 通用字段
	Replicas    *int32               `json:"replicas,omitempty"`
	Annotations map[string]string    `json:"annotations,omitempty"`
	
	// 平台特定字段（通过嵌入接口实现）
	ProviderSpec interface{} `json:"providerSpec,omitempty"`
}

// ProviderSpec 接口，各平台实现自己的 Spec
type ProviderSpec interface {
	Validate() error
	ToRawExtension() (runtime.RawExtension, error)
}

// 通用的 Machineset 列表
type MachineSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineSet `json:"items"`
}