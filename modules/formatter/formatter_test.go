package formatter

import (
	"log/slog"
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	formatter := Format[string](func(groups []string, key string, value slog.Value) slog.Value {
		return slog.StringValue("formatted_" + value.String())
	})

	attr := slog.String("test", "value")
	result, ok := formatter([]string{"group1"}, attr)

	if !ok {
		t.Error("Expected formatter to return true")
	}
	if result.String() != "formatted_value" {
		t.Errorf("Expected 'formatted_value', got %s", result.String())
	}
}

func TestFormatByType_String(t *testing.T) {
	formatter := FormatByType[string](func(s string) slog.Value {
		return slog.StringValue("processed_" + s)
	})

	attr := slog.String("test", "value")
	result, ok := formatter([]string{}, attr)

	if !ok {
		t.Error("Expected formatter to return true")
	}
	if result.String() != "processed_value" {
		t.Errorf("Expected 'processed_value', got %s", result.String())
	}
}

func TestFormatByType_Int64(t *testing.T) {
	formatter := FormatByType[int64](func(i int64) slog.Value {
		return slog.Int64Value(i * 2)
	})

	attr := slog.Int("number", 42)
	result, ok := formatter([]string{}, attr)

	if !ok {
		t.Error("Expected formatter to return true")
	}
	if result.Int64() != 84 {
		t.Errorf("Expected 84, got %d", result.Int64())
	}
}

func TestFormatByType_Group(t *testing.T) {
	formatter := FormatByType[string](func(s string) slog.Value {
		return slog.StringValue("formatted_" + s)
	})

	attr := slog.Group("test_group",
		slog.String("key", "value"),
		slog.Group("nested", slog.String("inner", "text")),
	)
	result, ok := formatter([]string{}, attr)

	if !ok {
		t.Fatal("expected formatter to update group values")
	}
	if result.Kind() != slog.KindGroup {
		t.Fatalf("expected group value, got kind %v", result.Kind())
	}

	group := result.Group()
	if len(group) != 2 {
		t.Fatalf("expected two attributes in group, got %d", len(group))
	}
	if group[0].Value.String() != "formatted_value" {
		t.Fatalf("expected formatted top-level value, got %s", group[0].Value.String())
	}
	nested := group[1].Value.Group()
	if len(nested) != 1 {
		t.Fatalf("expected single nested attribute, got %d", len(nested))
	}
	if nested[0].Value.String() != "formatted_text" {
		t.Fatalf("expected formatted nested value, got %s", nested[0].Value.String())
	}
}

func TestFormatByType_WrongType(t *testing.T) {
	formatter := FormatByType[string](func(s string) slog.Value {
		return slog.StringValue("should_not_be_called")
	})

	attr := slog.Int("number", 42)
	result, ok := formatter([]string{}, attr)

	if ok {
		t.Error("Expected formatter to return false for wrong type")
	}
	if result.Int64() != 42 {
		t.Error("Expected original value to be returned unchanged")
	}
}

func TestFormatByType_Time(t *testing.T) {
	now := time.Now()
	formatter := FormatByType[time.Time](func(t time.Time) slog.Value {
		return slog.StringValue(t.Format("2006-01-02"))
	})

	attr := slog.Time("timestamp", now)
	result, ok := formatter([]string{}, attr)

	if !ok {
		t.Error("Expected formatter to return true")
	}
	expected := now.Format("2006-01-02")
	if result.String() != expected {
		t.Errorf("Expected %s, got %s", expected, result.String())
	}
}

func TestFormatByKind(t *testing.T) {
	formatter := FormatByKind(slog.KindString, func(value slog.Value) slog.Value {
		return slog.StringValue("kind_" + value.String())
	})

	attr := slog.String("test", "value")
	result, ok := formatter([]string{}, attr)

	if !ok {
		t.Error("Expected formatter to return true")
	}
	if result.String() != "kind_value" {
		t.Errorf("Expected 'kind_value', got %s", result.String())
	}
}

func TestFormatByKind_WrongKind(t *testing.T) {
	formatter := FormatByKind(slog.KindString, func(value slog.Value) slog.Value {
		return slog.StringValue("should_not_be_called")
	})

	attr := slog.Int("number", 42)
	result, ok := formatter([]string{}, attr)

	if ok {
		t.Error("Expected formatter to return false for wrong kind")
	}
	if result.Int64() != 42 {
		t.Error("Expected original value to be returned unchanged")
	}
}

func TestFormatByKind_Group(t *testing.T) {
	formatter := FormatByKind(slog.KindString, func(value slog.Value) slog.Value {
		return slog.StringValue("formatted_" + value.String())
	})

	// 创建包含字符串属性的组
	attr := slog.Group("test_group",
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	)

	result, ok := formatter([]string{}, attr)

	if !ok {
		t.Error("Expected formatter to return true for group with matching kind")
	}

	// 检查组中的字符串是否被格式化
	group := result.Group()
	if len(group) != 2 {
		t.Errorf("Expected 2 attributes in group, got %d", len(group))
	}

	// 第一个属性应该被格式化
	if group[0].Value.String() != "formatted_value1" {
		t.Errorf("Expected 'formatted_value1', got %s", group[0].Value.String())
	}

	// 第二个属性应该保持不变
	if group[1].Value.Int64() != 42 {
		t.Errorf("Expected 42, got %d", group[1].Value.Int64())
	}
}
