package common

// ToAnySlice returns a slice with all elements mapped to `any` type
func ToAnySlice[T any](collection []T) []any {
	result := make([]any, len(collection))
	for i := range collection {
		result[i] = collection[i]
	}
	return result
}

// MapToSlice transforms a map into a slice based on specific iteratee
// Play: https://go.dev/play/p/ZuiCZpDt6LD
func MapToSlice[K comparable, V any, R any](in map[K]V, iteratee func(key K, value V) R) []R {
	result := make([]R, 0, len(in))

	for k := range in {
		result = append(result, iteratee(k, in[k]))
	}

	return result
}

// Contains returns true if an element is present in a collection.
func Contains[T comparable](collection []T, element T) bool {
	for i := range collection {
		if collection[i] == element {
			return true
		}
	}

	return false
}
