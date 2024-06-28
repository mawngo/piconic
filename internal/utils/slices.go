package utils

// Uniq returns a duplicate-free version of an array, in which only the first occurrence of each element is kept.
// The order of result values is determined by the order they occur in the array.
// Play: https://go.dev/play/p/DTzbeXZ6iEN
func Uniq[T comparable](collection []T) []T {
	result := make([]T, 0, len(collection))
	seen := make(map[T]struct{}, len(collection))

	for i := range collection {
		if _, ok := seen[collection[i]]; ok {
			continue
		}

		seen[collection[i]] = struct{}{}
		result = append(result, collection[i])
	}

	return result
}
