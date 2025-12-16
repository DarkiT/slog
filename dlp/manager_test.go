package dlp

import (
	"fmt"
	"testing"
)

func TestDesensitizerManager_Registration(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 验证初始化后已有脱敏器
	desensitizers := manager.ListDesensitizers()
	if len(desensitizers) == 0 {
		t.Error("SecurityEnhancedManager should have default desensitizers registered")
	}

	// 测试重复注册应该失败
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	err := manager.RegisterDesensitizer(phoneDesensitizer)
	if err == nil {
		t.Error("Should not allow duplicate registration")
	}

	// 测试获取脱敏器
	retrieved, exists := manager.GetDesensitizer("phone")
	if !exists {
		t.Error("Should find registered desensitizer")
	}
	if retrieved.Name() != "phone" {
		t.Errorf("Retrieved desensitizer name = %s, want phone", retrieved.Name())
	}

	// 测试注销脱敏器
	err = manager.UnregisterDesensitizer("phone")
	if err != nil {
		t.Errorf("UnregisterDesensitizer() error = %v", err)
	}

	// 验证注销成功
	_, exists = manager.GetDesensitizer("phone")
	if exists {
		t.Error("Desensitizer should be unregistered")
	}
}

func TestDesensitizerManager_TypeMapping(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 注册多个脱敏器
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	emailDesensitizer := NewEnhancedEmailDesensitizer()

	manager.RegisterDesensitizer(phoneDesensitizer)
	manager.RegisterDesensitizer(emailDesensitizer)

	// 测试类型映射
	phoneDesensitizers := manager.GetDesensitizersForType("phone")
	if len(phoneDesensitizers) != 1 {
		t.Errorf("GetDesensitizersForType(phone) length = %d, want 1", len(phoneDesensitizers))
	}

	emailDesensitizers := manager.GetDesensitizersForType("email")
	if len(emailDesensitizers) != 1 {
		t.Errorf("GetDesensitizersForType(email) length = %d, want 1", len(emailDesensitizers))
	}

	// 测试不存在的类型
	unknownDesensitizers := manager.GetDesensitizersForType("unknown")
	if unknownDesensitizers != nil {
		t.Error("Should return nil for unknown type")
	}

	// 测试类型映射详情
	mapping := manager.GetTypeMapping()
	if len(mapping["phone"]) != 1 || mapping["phone"][0] != "phone" {
		t.Error("Type mapping should contain phone -> [phone]")
	}
	if len(mapping["email"]) != 1 || mapping["email"][0] != "email" {
		t.Error("Type mapping should contain email -> [email]")
	}
}

func TestDesensitizerManager_Processing(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 注册脱敏器
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	emailDesensitizer := NewEnhancedEmailDesensitizer()

	manager.RegisterDesensitizer(phoneDesensitizer)
	manager.RegisterDesensitizer(emailDesensitizer)

	// 测试指定脱敏器处理
	result, err := manager.ProcessWithDesensitizer("phone", "13812345678")
	if err != nil {
		t.Errorf("ProcessWithDesensitizer() error = %v", err)
	}
	if result.Desensitized != "138****5678" {
		t.Errorf("ProcessWithDesensitizer() result = %s, want 138****5678", result.Desensitized)
	}
	if result.Desensitizer != "phone" {
		t.Errorf("ProcessWithDesensitizer() desensitizer = %s, want phone", result.Desensitizer)
	}

	// 测试按类型处理
	result, err = manager.ProcessWithType("email", "test@example.com")
	if err != nil {
		t.Errorf("ProcessWithType() error = %v", err)
	}
	if result.Desensitized != "t**@example.com" {
		t.Errorf("ProcessWithType() result = %s, want t**@example.com", result.Desensitized)
	}

	// 测试自动检测处理
	result, err = manager.AutoDetectAndProcess("13812345678")
	if err != nil {
		t.Errorf("AutoDetectAndProcess() error = %v", err)
	}
	if result.Desensitized != "138****5678" {
		t.Errorf("AutoDetectAndProcess() result = %s, want 138****5678", result.Desensitized)
	}
}

func TestDesensitizerManager_UpsertAndVersion(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	versionBefore := manager.CurrentVersion()

	phone := NewEnhancedPhoneDesensitizer()
	v1, err := manager.UpsertDesensitizer(phone)
	if err != nil {
		t.Fatalf("UpsertDesensitizer error: %v", err)
	}
	if v1 <= versionBefore {
		t.Fatalf("version should increase, got %d, before %d", v1, versionBefore)
	}

	replacer := &dummyDesensitizer{name: "phone"}
	v2, err := manager.UpsertDesensitizer(replacer)
	if err != nil {
		t.Fatalf("UpsertDesensitizer replace error: %v", err)
	}
	if v2 <= v1 {
		t.Fatalf("version should increase after replace, got %d, prev %d", v2, v1)
	}

	got, ok := manager.GetDesensitizer("phone")
	if !ok {
		t.Fatal("expected phone desensitizer after replace")
	}
	if got == phone {
		t.Fatal("expected replacement to take effect")
	}
	stats := manager.GetStats()
	if stats.Version != v2 {
		t.Fatalf("stats version mismatch, want %d, got %d", v2, stats.Version)
	}
}

type dummyDesensitizer struct {
	name string
}

func (d *dummyDesensitizer) Name() string                            { return d.name }
func (d *dummyDesensitizer) Supports(_ string) bool                  { return true }
func (d *dummyDesensitizer) Desensitize(data string) (string, error) { return data, nil }
func (d *dummyDesensitizer) Configure(map[string]interface{}) error  { return nil }
func (d *dummyDesensitizer) Enabled() bool                           { return true }
func (d *dummyDesensitizer) Enable()                                 {}
func (d *dummyDesensitizer) Disable()                                {}

func TestDesensitizerManager_Stats(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 注册脱敏器（SecurityEnhancedManager已自动注册了默认脱敏器）
	// 获取已注册的脱敏器数量
	initialCount := len(manager.ListDesensitizers())

	// 获取统计信息
	stats := manager.GetStats()
	if stats.TotalDesensitizers != initialCount {
		t.Errorf("TotalDesensitizers = %d, want %d", stats.TotalDesensitizers, initialCount)
	}
	if stats.EnabledDesensitizers != initialCount {
		t.Errorf("EnabledDesensitizers = %d, want %d", stats.EnabledDesensitizers, initialCount)
	}

	// 检查类型覆盖（仅检查增强版支持的类型）
	expectedTypes := []string{"phone", "mobile", "mobile_phone", "email", "mail", "email_address"}
	for _, dataType := range expectedTypes {
		if stats.TypeCoverage[dataType] < 1 {
			t.Errorf("Type %s should be covered", dataType)
		}
	}

	// 执行一些处理以更新性能指标
	manager.ProcessWithDesensitizer("phone", "13812345678")
	manager.ProcessWithDesensitizer("email", "test@example.com")

	// 获取更新后的统计信息
	updatedStats := manager.GetStats()
	if updatedStats.PerformanceMetrics["phone"].TotalCalls < 1 {
		t.Error("Performance metrics should be updated")
	}
}

func TestDesensitizerManager_EnableDisable(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 注册脱敏器
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	emailDesensitizer := NewEnhancedEmailDesensitizer()

	manager.RegisterDesensitizer(phoneDesensitizer)
	manager.RegisterDesensitizer(emailDesensitizer)

	// 测试禁用所有脱敏器
	manager.DisableAll()
	stats := manager.GetStats()
	if stats.EnabledDesensitizers != 0 {
		t.Errorf("EnabledDesensitizers after DisableAll = %d, want 0", stats.EnabledDesensitizers)
	}

	// 验证处理时不脱敏
	result, err := manager.ProcessWithDesensitizer("phone", "13812345678")
	if err != nil {
		t.Errorf("ProcessWithDesensitizer() error = %v", err)
	}
	if result.Desensitized != "13812345678" {
		t.Error("Disabled desensitizer should return original text")
	}

	// 测试启用所有脱敏器
	manager.EnableAll()
	stats = manager.GetStats()
	initialCount := len(manager.ListDesensitizers())
	if stats.EnabledDesensitizers != initialCount {
		t.Errorf("EnabledDesensitizers after EnableAll = %d, want %d", stats.EnabledDesensitizers, initialCount)
	}

	// 验证处理时正常脱敏
	result, err = manager.ProcessWithDesensitizer("phone", "13812345678")
	if err != nil {
		t.Errorf("ProcessWithDesensitizer() error = %v", err)
	}
	if result.Desensitized != "138****5678" {
		t.Error("Enabled desensitizer should desensitize text")
	}
}

func TestDesensitizerManager_Cache(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 注册脱敏器
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	manager.RegisterDesensitizer(phoneDesensitizer)

	input := "13812345678"

	// 第一次处理
	result1, err := manager.ProcessWithDesensitizer("phone", input)
	if err != nil {
		t.Errorf("First processing error = %v", err)
	}

	// 第二次处理（应使用缓存）
	result2, err := manager.ProcessWithDesensitizer("phone", input)
	if err != nil {
		t.Errorf("Second processing error = %v", err)
	}

	if result1.Desensitized != result2.Desensitized {
		t.Error("Results should be identical")
	}

	// 第二次处理应该更快（来自缓存）
	if result2.Duration > result1.Duration*10 { // 允许一定的时间差异
		t.Error("Second call should be faster (cached)")
	}

	// 测试清除缓存
	manager.ClearAllCaches()

	// 验证缓存统计被重置
	detailedStats := manager.GetDetailedStats()
	cacheStats := detailedStats["cache_stats"].(map[string]CacheStats)
	if stats, exists := cacheStats["phone"]; exists && stats.Hits > 0 {
		t.Error("Cache should be cleared")
	}
}

func TestDesensitizerManager_ErrorHandling(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 测试注册nil脱敏器
	err := manager.RegisterDesensitizer(nil)
	if err == nil {
		t.Error("Should not allow nil desensitizer registration")
	}

	// 测试获取不存在的脱敏器
	_, exists := manager.GetDesensitizer("nonexistent")
	if exists {
		t.Error("Should not find nonexistent desensitizer")
	}

	// 测试使用不存在的脱敏器处理
	_, err = manager.ProcessWithDesensitizer("nonexistent", "test")
	if err == nil {
		t.Error("Should return error for nonexistent desensitizer")
	}

	// 测试注销不存在的脱敏器
	err = manager.UnregisterDesensitizer("nonexistent")
	if err == nil {
		t.Error("Should return error when unregistering nonexistent desensitizer")
	}

	// 测试使用不支持的类型处理
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	manager.RegisterDesensitizer(phoneDesensitizer)

	_, err = manager.ProcessWithType("unsupported_type", "test")
	if err == nil {
		t.Error("Should return error for unsupported type")
	}
}

func TestDesensitizerManager_ManagerEnableDisable(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 注册脱敏器
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	manager.RegisterDesensitizer(phoneDesensitizer)

	// 测试禁用管理器
	manager.Disable()
	if manager.IsEnabled() {
		t.Error("Manager should be disabled")
	}

	// 禁用的管理器应该返回原文
	result, err := manager.AutoDetectAndProcess("13812345678")
	if err == nil || result.Desensitized != "13812345678" {
		t.Errorf("Disabled manager should return original text with error, got: %s", result.Desensitized)
	}

	// 测试启用管理器
	manager.Enable()
	if !manager.IsEnabled() {
		t.Error("Manager should be enabled")
	}

	// 启用的管理器应该正常工作
	result, err = manager.AutoDetectAndProcess("13812345678")
	if err != nil {
		t.Errorf("Enabled manager should work, error = %v", err)
	}
	if result.Desensitized != "138****5678" {
		t.Errorf("Enabled manager should desensitize, got: %s", result.Desensitized)
	}
}

func TestDesensitizerManager_PerformanceMetrics(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 注册脱敏器
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	manager.RegisterDesensitizer(phoneDesensitizer)

	// 执行多次处理以积累指标
	inputs := []string{"13812345678", "invalid", "15987654321"}
	for _, input := range inputs {
		manager.ProcessWithDesensitizer("phone", input)
	}

	// 获取性能指标
	stats := manager.GetStats()
	phoneMetrics := stats.PerformanceMetrics["phone"]

	if phoneMetrics.TotalCalls != 3 {
		t.Errorf("TotalCalls = %d, want 3", phoneMetrics.TotalCalls)
	}

	if phoneMetrics.AverageDuration <= 0 {
		t.Error("AverageDuration should be positive")
	}

	if phoneMetrics.SuccessRate <= 0 {
		t.Error("SuccessRate should be positive")
	}

	// 测试详细统计信息
	detailedStats := manager.GetDetailedStats()
	if detailedStats["total_desensitizers"] == nil || detailedStats["total_desensitizers"].(int) == 0 {
		t.Error("Detailed stats should include total_desensitizers")
	}

	if detailedStats["manager_enabled"] != true {
		t.Error("Detailed stats should show manager as enabled")
	}
}

func TestDesensitizerManager_Concurrency(t *testing.T) {
	manager := NewSecurityEnhancedManager()

	// 注册脱敏器
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	manager.RegisterDesensitizer(phoneDesensitizer)

	// 并发测试
	numRoutines := 50
	results := make(chan *DesensitizationResult, numRoutines)

	for i := 0; i < numRoutines; i++ {
		go func() {
			result, err := manager.ProcessWithDesensitizer("phone", "13812345678")
			if err != nil {
				t.Errorf("Concurrent processing error: %v", err)
			}
			results <- result
		}()
	}

	// 验证所有结果
	for i := 0; i < numRoutines; i++ {
		result := <-results
		if result.Desensitized != "138****5678" {
			t.Errorf("Concurrent result = %s, want 138****5678", result.Desensitized)
		}
	}

	// 验证统计信息正确性
	stats := manager.GetStats()
	if stats.PerformanceMetrics["phone"].TotalCalls != int64(numRoutines) {
		t.Errorf("TotalCalls = %d, want %d", stats.PerformanceMetrics["phone"].TotalCalls, numRoutines)
	}
}

// 性能基准测试

func BenchmarkDesensitizerManager_Register(b *testing.B) {
	manager := NewSecurityEnhancedManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 创建唯一名称的脱敏器避免重复注册错误
		desensitizer := NewEnhancedPhoneDesensitizer()
		desensitizer.name = fmt.Sprintf("phone_%d", i)
		manager.RegisterDesensitizer(desensitizer)
	}
}

func BenchmarkDesensitizerManager_ProcessWithDesensitizer(b *testing.B) {
	manager := NewSecurityEnhancedManager()
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	manager.RegisterDesensitizer(phoneDesensitizer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ProcessWithDesensitizer("phone", "13812345678")
	}
}

func BenchmarkDesensitizerManager_AutoDetectAndProcess(b *testing.B) {
	manager := NewSecurityEnhancedManager()
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	emailDesensitizer := NewEnhancedEmailDesensitizer()
	manager.RegisterDesensitizer(phoneDesensitizer)
	manager.RegisterDesensitizer(emailDesensitizer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.AutoDetectAndProcess("13812345678")
	}
}

func BenchmarkDesensitizerManager_GetStats(b *testing.B) {
	manager := NewSecurityEnhancedManager()
	phoneDesensitizer := NewEnhancedPhoneDesensitizer()
	manager.RegisterDesensitizer(phoneDesensitizer)

	// 添加一些历史数据
	for i := 0; i < 100; i++ {
		manager.ProcessWithDesensitizer("phone", "13812345678")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GetStats()
	}
}
