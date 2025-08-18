package ptr

// Ptc return a pointer to value
func Ptr[T any](value T) *T {
	return &value
}
