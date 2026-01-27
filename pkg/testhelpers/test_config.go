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

package testhelpers

import (
	"os"
	"strings"
)

// TestConfig contains configuration for E2E tests.
type TestConfig struct {
	// Backend specifies which API backend to test against (mapi or capi).
	Backend BackendType

	// AuthoritativeAPI specifies which API is authoritative.
	// This is useful for testing migration scenarios where MAPI backend
	// is used but CAPI is the authoritative source.
	AuthoritativeAPI BackendType
}

// GetTestConfig returns the test configuration based on environment variables.
//
// Environment variables:
//   - TEST_BACKEND: "mapi" or "capi" (default: "mapi")
//   - TEST_AUTHORITY: "mapi" or "capi" (default: same as TEST_BACKEND)
//
// Examples:
//   - TEST_BACKEND=mapi TEST_AUTHORITY=mapi  -> Pure MAPI testing (traditional)
//   - TEST_BACKEND=capi TEST_AUTHORITY=capi  -> Pure CAPI testing (future)
//   - TEST_BACKEND=mapi TEST_AUTHORITY=capi  -> MAPI backend + CAPI authority (migration scenario)
//
// Migration path (2030):
//  1. Remove TEST_BACKEND environment variable handling
//  2. Always return BackendCAPI
//  3. Remove AuthoritativeAPI field entirely
func GetTestConfig() *TestConfig {
	backend := parseBackendType(os.Getenv("TEST_BACKEND"))
	authority := parseBackendType(os.Getenv("TEST_AUTHORITY"))

	// If authority is not specified, default to backend
	if authority == "" {
		authority = backend
	}

	// If backend is not specified, default to MAPI for backwards compatibility
	if backend == "" {
		backend = BackendMAPI
		authority = BackendMAPI
	}

	return &TestConfig{
		Backend:          backend,
		AuthoritativeAPI: authority,
	}
}

// parseBackendType converts a string to BackendType.
func parseBackendType(s string) BackendType {
	switch strings.ToLower(s) {
	case "mapi", "machineapi":
		return BackendMAPI
	case "capi", "clusterapi":
		return BackendCAPI
	default:
		return ""
	}
}

// IsMigrationScenario returns true if testing MAPI backend with CAPI authority.
// This represents the migration period where CAPI is authoritative but MAPI
// resources still exist (synced from CAPI).
func (tc *TestConfig) IsMigrationScenario() bool {
	return tc.Backend == BackendMAPI && tc.AuthoritativeAPI == BackendCAPI
}

// IsPureMAPI returns true if testing pure MAPI (pre-migration).
func (tc *TestConfig) IsPureMAPI() bool {
	return tc.Backend == BackendMAPI && tc.AuthoritativeAPI == BackendMAPI
}

// IsPureCAPI returns true if testing pure CAPI (post-migration).
func (tc *TestConfig) IsPureCAPI() bool {
	return tc.Backend == BackendCAPI && tc.AuthoritativeAPI == BackendCAPI
}
