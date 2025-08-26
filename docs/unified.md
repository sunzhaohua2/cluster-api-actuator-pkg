# Unified MAPI and CAPI Test Framework

## Overview

This unified test framework provides an abstraction layer that allows the same test suite to run against both Machine API (MAPI) and Cluster API (CAPI) backends without requiring complete reimplementation. The framework supports testing the OpenShift conversion layer functionality, enabling comprehensive validation of API migration scenarios.

## Architecture

### Core Components

The unified framework consists of several key components:

1. **MachineBackend Interface** - Abstract interface defining common operations
2. **MAPI Backend Implementation** - MAPI-specific operations
3. **CAPI Backend Implementation** - CAPI-specific operations
4. **Unified Test Framework** - Orchestrates testing across different backends
5. **Test Configuration Management** - Environment-based configuration system

### Backend Types and Authoritative APIs

- **Backend Type**: Determines which client/resources the test code uses (MAPI or CAPI)
- **Authoritative API**: Test framework configuration parameter that determines what value to set in the MachineSet's `authoritativeAPI` field

## Configuration

### Environment Variables

Configure the test framework using these environment variables:

```bash
# Set backend type (MAPI or CAPI)
export TEST_BACKEND_TYPE=MAPI

# Set authoritative API type (MAPI or CAPI)
export TEST_AUTHORITATIVE_API=MAPI
```

### Supported Test Scenarios

The framework supports three main testing scenarios:

1. **MAPI Authoritative Testing**
   ```bash
   TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=MAPI
   ```
   - Uses MAPI backend to create MAPI MachineSets with `authoritativeAPI: MachineAPI`
   - MAPI serves as the authoritative API for machine lifecycle management
   - OpenShift automatically creates corresponding CAPI mirror resources
   - Machine lifecycle is controlled by MAPI controllers

2. **Pure CAPI Testing**
   ```bash
   TEST_BACKEND_TYPE=CAPI TEST_AUTHORITATIVE_API=CAPI
   ```
   - Uses CAPI backend to create CAPI MachineSets
   - CAPI serves as the authoritative API
   - Pure CAPI behavior

3. **CAPI Authoritative Testing**
   ```bash
   TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=CAPI
   ```
   - Test code uses MAPI backend to create MAPI MachineSets with `authoritativeAPI: ClusterAPI`
   - CAPI serves as the authoritative API for machine lifecycle management
   - Machine lifecycle management is transferred to CAPI controllers

## Usage

### Using Makefile Targets (Recommended)

The `Makefile.test` provides convenient targets for different test scenarios:

```bash
# MAPI Authoritative testing: MAPI backend + MAPI authoritative
make -f Makefile.test test-mapi

# Pure CAPI testing: CAPI backend + CAPI authoritative
make -f Makefile.test test-capi

# CAPI Authoritative testing: MAPI backend + CAPI authoritative
make -f Makefile.test test-mapi-with-capi-auth

# Run all test configurations
make -f Makefile.test test-all

# Show current configuration
make -f Makefile.test show-config

# Display help
make -f Makefile.test help
```

### Direct go test Usage

```bash
# MAPI Authoritative testing
TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=MAPI go test -v ./pkg/unified/ -ginkgo.v

# Pure CAPI testing
TEST_BACKEND_TYPE=CAPI TEST_AUTHORITATIVE_API=CAPI go test -v ./pkg/unified/ -ginkgo.v

# CAPI Authoritative testing
TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=CAPI go test -v ./pkg/unified/ -ginkgo.v
```

## Available Test Targets

| Target | Description |
|--------|-------------|
| `test-mapi` | Run tests with MAPI backend and MAPI authoritative API |
| `test-capi` | Run tests with CAPI backend and CAPI authoritative API |
| `test-mapi-with-capi-auth` | Run tests with MAPI backend but CAPI authoritative API |
| `test-all` | Run all supported test configurations |
| `show-config` | Show current test configuration |
| `help` | Display help information |

## Benefits

1. **Code Reuse**: Write tests once, run against multiple backends
2. **Conversion Testing**: Validate OpenShift conversion layer functionality
3. **Flexible Configuration**: Easy switching between test modes
4. **Comprehensive Coverage**: Test both pure API scenarios and conversion layers

## Development Guidelines

### Adding New Tests

1. Use the `UnifiedFramework` interface instead of direct API calls
2. Write backend-agnostic test logic
3. Verify configurations using the flexible validation helper functions VerifyMachineSetContainsString
