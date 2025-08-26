package framework

import (
	"fmt"
	"os"
	"strings"
)

// TestConfig defines test configuration
type TestConfig struct {
	// Backend type: MAPI or CAPI
	BackendType BackendType

	// Authoritative API type: MAPI or CAPI
	AuthoritativeAPI BackendType
}

// LoadTestConfig loads test configuration from environment variables
func LoadTestConfig() *TestConfig {
	config := &TestConfig{
		BackendType:      BackendTypeMAPI, // Default to MAPI
		AuthoritativeAPI: BackendTypeMAPI, // Default to MAPI
	}

	// Read configuration from environment variables
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

	return config
}

// GetBackend returns backend instance based on configuration
func (tc *TestConfig) GetBackend() (MachineBackend, error) {
	factory := &BackendFactory{}
	return factory.NewBackend(tc.BackendType, tc.AuthoritativeAPI)
}

// IsMAPIBackend checks if this is a MAPI backend
func (tc *TestConfig) IsMAPIBackend() bool {
	return tc.BackendType == BackendTypeMAPI
}

// IsCAPIBackend checks if this is a CAPI backend
func (tc *TestConfig) IsCAPIBackend() bool {
	return tc.BackendType == BackendTypeCAPI
}

// GetTestDescription returns test description including backend information
func (tc *TestConfig) GetTestDescription() string {
	if tc.BackendType != tc.AuthoritativeAPI {
		return fmt.Sprintf("Backend: %s, AuthoritativeAPI: %s",
			tc.BackendType, tc.AuthoritativeAPI)
	}
	return fmt.Sprintf("Backend: %s", tc.BackendType)
}
