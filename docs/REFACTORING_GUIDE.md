# 重构指南：统一MAPI和CAPI测试框架

## 概述

本重构的目标是创建一个抽象接口，让测试可以同时支持MAPI和CAPI后端，而无需完全重新实现。我们还支持设置权威API类型来测试转换层功能。

## 架构设计

### 核心组件

1. **MachineBackend接口** (`pkg/framework/backend_interface.go`)
   - 定义机器后端的抽象接口
   - 支持MAPI和CAPI两种后端类型
   - 支持设置权威API类型

2. **MAPI后端实现** (`pkg/framework/mapi_backend.go`)
   - 实现MAPI特定的机器操作
   - 支持通过转换层创建CAPI资源

3. **CAPI后端实现** (`pkg/framework/capi_backend.go`)
   - 实现CAPI特定的机器操作
   - 支持通过转换层创建MAPI资源

4. **统一测试框架** (`pkg/framework/unified_framework.go`)
   - 提供统一的测试接口
   - 根据配置自动选择后端
   - 支持转换层测试

5. **测试配置管理** (`pkg/framework/test_config.go`)
   - 从环境变量加载测试配置
   - 支持动态配置后端类型和权威API

## 使用方法

### 环境变量配置

```bash
# 设置后端类型
export TEST_BACKEND_TYPE=MAPI        # 或 CAPI

# 设置权威API类型
export TEST_AUTHORITATIVE_API=MAPI   # 或 CAPI

# 启用转换层测试（当后端类型和权威API不同时）
export TEST_BACKEND_TYPE=MAPI
export TEST_AUTHORITATIVE_API=CAPI
```

### 运行测试

#### 使用Makefile（推荐）

```bash
# 运行MAPI后端测试
make -f Makefile.test test-mapi

# 运行CAPI后端测试
make -f Makefile.test test-capi

# 测试转换层：MAPI后端 + CAPI权威API
make -f Makefile.test test-mapi-with-capi-auth

# 测试转换层：CAPI后端 + MAPI权威API
make -f Makefile.test test-capi-with-mapi-auth

# 运行所有测试配置
make -f Makefile.test test-all

# 显示帮助信息
make -f Makefile.test help
```

#### 直接使用go test

```bash
# MAPI后端，MAPI权威API
TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=MAPI go test -v ./cluster-capi-operator/e2e/

# CAPI后端，CAPI权威API
TEST_BACKEND_TYPE=CAPI TEST_AUTHORITATIVE_API=CAPI go test -v ./cluster-capi-operator/e2e/

# MAPI后端，CAPI权威API（测试转换层）
TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=CAPI go test -v ./cluster-capi-operator/e2e/
```

### 在测试代码中使用

```go
package e2e

import (
    "github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
)

var _ = Describe("Unified Tests", func() {
    var unifiedFramework *framework.UnifiedFramework
    
    BeforeAll(func() {
        var err error
        unifiedFramework, err = framework.NewUnifiedFramework()
        Expect(err).NotTo(HaveOccurred())
        
        // 输出配置信息
        GinkgoWriter.Printf("Backend: %s\n", unifiedFramework.GetBackendType())
        GinkgoWriter.Printf("Authoritative API: %s\n", unifiedFramework.GetAuthoritativeAPI())
        GinkgoWriter.Printf("Conversion Layer: %t\n", unifiedFramework.IsConversionLayerEnabled())
    })
    
    It("should work with any backend", func() {
        // 使用统一接口，无需关心具体后端
        machineSet, err := unifiedFramework.CreateMachineSet(ctx, client, params)
        Expect(err).NotTo(HaveOccurred())
        
        // 测试逻辑...
    })
})
```

## 测试场景

### 1. 纯MAPI测试
```bash
TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=MAPI
```
- 使用MAPI后端
- 直接操作MAPI资源
- 不涉及转换层

### 2. 纯CAPI测试
```bash
TEST_BACKEND_TYPE=CAPI TEST_AUTHORITATIVE_API=CAPI
```
- 使用CAPI后端
- 直接操作CAPI资源
- 不涉及转换层

### 3. 转换层测试：MAPI后端 + CAPI权威API
```bash
TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=CAPI
```
- 使用MAPI后端管理资源
- 但通过转换层创建CAPI资源
- 测试MAPI到CAPI的转换

### 4. 转换层测试：CAPI后端 + MAPI权威API
```bash
TEST_BACKEND_TYPE=CAPI TEST_AUTHORITATIVE_API=MAPI
```
- 使用CAPI后端管理资源
- 但通过转换层创建MAPI资源
- 测试CAPI到MAPI的转换

## 扩展指南

### 添加新的后端类型

1. 在`BackendType`中添加新类型
2. 实现`MachineBackend`接口
3. 在`BackendFactory`中添加创建逻辑

### 添加新的平台支持

1. 在相应的后端实现中添加平台特定的逻辑
2. 实现平台特定的机器模板创建
3. 实现平台特定的机器规格获取

### 实现转换层

1. 在`createMachineSetWithConversion`方法中实现具体转换逻辑
2. 使用webhook或控制器进行资源转换
3. 验证转换后的资源状态

## 注意事项

1. **资源清理**：确保测试完成后正确清理所有创建的资源
2. **命名空间**：MAPI和CAPI使用不同的命名空间
3. **错误处理**：转换层可能失败，需要适当的错误处理
4. **平台兼容性**：某些功能可能只在特定平台上可用

## 故障排除

### 常见问题

1. **转换层未实现**：检查是否启用了转换层测试
2. **资源创建失败**：检查后端类型和权威API配置
3. **平台不支持**：确认当前平台是否支持所需功能

### 调试技巧

1. 使用`show-config`目标查看当前配置
2. 检查测试输出中的配置信息
3. 使用`-ginkgo.v`标志获取详细输出

## 未来改进

1. **完整的转换层实现**：实现MAPI和CAPI之间的双向转换
2. **更多平台支持**：扩展到其他云平台
3. **性能优化**：优化资源创建和删除操作
4. **监控和指标**：添加测试执行指标收集
