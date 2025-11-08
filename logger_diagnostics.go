package slog

import "github.com/darkit/slog/modules"

// ModuleDiagnostics 描述模块健康与指标信息。
type ModuleDiagnostics struct {
	Name     string             `json:"name"`
	Type     modules.ModuleType `json:"type"`
	Enabled  bool               `json:"enabled"`
	Healthy  *bool              `json:"healthy,omitempty"`
	Metrics  map[string]any     `json:"metrics,omitempty"`
	Priority int                `json:"priority"`
}

// CollectModuleDiagnostics 聚合已注册模块的健康状态与指标。
func CollectModuleDiagnostics() []ModuleDiagnostics {
	if ext == nil {
		return nil
	}
	modulesSnapshot := ext.snapshotModules()

	diags := make([]ModuleDiagnostics, 0, len(modulesSnapshot))
	for _, m := range modulesSnapshot {
		diag := ModuleDiagnostics{
			Name:     m.Name(),
			Type:     m.Type(),
			Enabled:  m.Enabled(),
			Priority: m.Priority(),
		}
		if healthable, ok := m.(modules.Healthable); ok {
			err := healthable.HealthCheck()
			healthy := err == nil && healthable.IsHealthy()
			diag.Healthy = &healthy
		}
		if measurable, ok := m.(modules.Measurable); ok {
			diag.Metrics = measurable.GetMetrics()
		}
		diags = append(diags, diag)
	}
	return diags
}
