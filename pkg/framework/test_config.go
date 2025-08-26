package framework

import (
	"fmt"
	"os"
	"strings"
)

// TestConfig 定义测试配置
type TestConfig struct {
	// 后端类型：MAPI 或 CAPI
	BackendType BackendType
	
	// 权威API类型：用于测试转换层
	AuthoritativeAPI BackendType
	
	// 是否启用转换层测试
	EnableConversionLayer bool
}

// LoadTestConfig 从环境变量加载测试配置
func LoadTestConfig() *TestConfig {
	config := &TestConfig{
		BackendType:      BackendTypeMAPI, // 默认使用MAPI
		AuthoritativeAPI: BackendTypeMAPI, // 默认权威API也是MAPI
	}
	
	// 从环境变量读取配置
	if backendType := os.Getenv("TEST_BACKEND_TYPE"); backendType != "" {
		switch strings.ToUpper(backendType) {
		case "MAPI":
			config.BackendType = BackendTypeMAPI
		case "CAPI":
			config.BackendType = BackendTypeCAPI
		}
	}
	
	if authAPI := os.Getenv("TEST_AUTHORITATIVE_API"); authAPI != "" {
		switch strings.ToUpper(authAPI) {
		case "MAPI":
			config.AuthoritativeAPI = BackendTypeMAPI
		case "CAPI":
			config.AuthoritativeAPI = BackendTypeCAPI
		}
	}
	
	// 如果后端类型和权威API不同，则启用转换层测试
	config.EnableConversionLayer = config.BackendType != config.AuthoritativeAPI
	
	return config
}

// GetBackend 根据配置获取后端实例
func (tc *TestConfig) GetBackend() (MachineBackend, error) {
	factory := &BackendFactory{}
	return factory.NewBackend(tc.BackendType, tc.AuthoritativeAPI)
}

// IsMAPIBackend 检查是否为MAPI后端
func (tc *TestConfig) IsMAPIBackend() bool {
	return tc.BackendType == BackendTypeMAPI
}

// IsCAPIBackend 检查是否为CAPI后端
func (tc *TestConfig) IsCAPIBackend() bool {
	return tc.BackendType == BackendTypeCAPI
}

// IsConversionLayerEnabled 检查是否启用转换层测试
func (tc *TestConfig) IsConversionLayerEnabled() bool {
	return tc.EnableConversionLayer
}

// GetTestDescription 获取测试描述，包含后端信息
func (tc *TestConfig) GetTestDescription() string {
	if tc.EnableConversionLayer {
		return fmt.Sprintf("Backend: %s, AuthoritativeAPI: %s (Conversion Layer)", 
			tc.BackendType, tc.AuthoritativeAPI)
	}
	return fmt.Sprintf("Backend: %s", tc.BackendType)
}
