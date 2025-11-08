package formatter

import "log/slog"

type (
	LogValuerFunc func(any) (slog.Value, bool)
	Formatter     func(groups []string, attr slog.Attr) (slog.Value, bool)
)

// Format pass every attributes into a formatter.
func Format[T any](formatter func([]string, string, slog.Value) slog.Value) Formatter {
	return func(groups []string, attr slog.Attr) (slog.Value, bool) {
		return formatter(groups, attr.Key, attr.Value), true
	}
}

// FormatByType pass attributes matching generic type into a formatter.
func FormatByType[T any](formatter func(T) slog.Value) Formatter {
	return func(_ []string, attr slog.Attr) (slog.Value, bool) {
		return applyFormatByType(formatter, attr.Value)
	}
}

func applyFormatByType[T any](formatter func(T) slog.Value, value slog.Value) (slog.Value, bool) {
	if v, ok := value.Any().(T); ok {
		return formatter(v), true
	}

	if value.Kind() != slog.KindGroup {
		return value, false
	}

	group := value.Group()
	updated := false
	attrs := make([]slog.Attr, len(group))

	for i, child := range group {
		formattedValue, changed := applyFormatByType(formatter, child.Value)
		if changed {
			updated = true
		}
		attrs[i] = slog.Attr{Key: child.Key, Value: formattedValue}
	}

	if updated {
		return slog.GroupValue(attrs...), true
	}

	return value, false
}

// FormatByKind pass attributes matching `slog.Kind` into a formatter.
func FormatByKind(kind slog.Kind, formatter func(slog.Value) slog.Value) Formatter {
	return func(_ []string, attr slog.Attr) (slog.Value, bool) {
		value := attr.Value

		if value.Kind() == kind {
			return formatter(value), true
		}

		if value.Kind() == slog.KindGroup {
			updated := false
			attrs := []slog.Attr{}

			for _, attr := range value.Group() {
				if attr.Value.Kind() == kind {
					attrs = append(attrs, slog.Attr{Key: attr.Key, Value: formatter(attr.Value)})
					updated = true
				} else {
					attrs = append(attrs, attr)
				}
			}

			if updated {
				return slog.GroupValue(attrs...), true
			}
		}

		return value, false
	}
}

// FormatByKey pass attributes matching key into a formatter.
func FormatByKey(key string, formatter func(slog.Value) slog.Value) Formatter {
	return func(_ []string, attr slog.Attr) (slog.Value, bool) {
		value := attr.Value

		if attr.Key == key {
			return formatter(value), true
		}

		if value.Kind() == slog.KindGroup {
			updated := false
			attrs := []slog.Attr{}

			for _, attr := range value.Group() {
				if attr.Key == key {
					attrs = append(attrs, slog.Attr{Key: attr.Key, Value: formatter(attr.Value)})
					updated = true
				} else {
					attrs = append(attrs, attr)
				}
			}

			if updated {
				return slog.GroupValue(attrs...), true
			}
		}

		return value, false
	}
}

// FormatByFieldType pass attributes matching both key and generic type into a formatter.
func FormatByFieldType[T any](key string, formatter func(T) slog.Value) Formatter {
	return func(_ []string, attr slog.Attr) (slog.Value, bool) {
		value := attr.Value

		if value.Kind() == slog.KindGroup || attr.Key != key {
			return value, false
		}

		if v, ok := value.Any().(T); ok {
			return formatter(v), true
		}

		return value, false
	}
}

// FormatByGroup pass attributes under a group into a formatter.
func FormatByGroup(targetGroup []string, formatter func([]slog.Attr) slog.Value) Formatter {
	return func(currentGroup []string, attr slog.Attr) (slog.Value, bool) {
		value := attr.Value

		if value.Kind() != slog.KindGroup || !matchFullGroupPath(currentGroup, attr.Key, targetGroup) {
			return value, false
		}

		return formatter(value.Group()), true
	}
}

// FormatByGroupKey pass attributes under a group and matching key, into a formatter.
func FormatByGroupKey(targetGroup []string, key string, formatter func(slog.Value) slog.Value) Formatter {
	return func(currentGroup []string, attr slog.Attr) (slog.Value, bool) {
		value := attr.Value

		if !equalStringSlices(targetGroup, currentGroup) || attr.Key != key {
			return value, false
		}

		return formatter(value), true
	}
}

// FormatByGroupKeyType pass attributes under a group, matching key and matching a generic type, into a formatter.
func FormatByGroupKeyType[T any](targetGroup []string, key string, formatter func(T) slog.Value) Formatter {
	return func(currentGroup []string, attr slog.Attr) (slog.Value, bool) {
		value := attr.Value

		if value.Kind() == slog.KindGroup || !equalStringSlices(targetGroup, currentGroup) || attr.Key != key {
			return value, false
		}

		if v, ok := value.Any().(T); ok {
			return formatter(v), true
		}

		return value, false
	}
}

func matchFullGroupPath(current []string, key string, target []string) bool {
	if len(target) != len(current)+1 {
		return false
	}
	for i := 0; i < len(current); i++ {
		if current[i] != target[i] {
			return false
		}
	}
	return target[len(target)-1] == key
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
