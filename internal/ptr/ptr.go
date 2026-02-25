package ptr

// Ptc return a pointer to value
func Ptr[T any](value T) *T {
	return &value
}

// Value returns a value of given pointer
func Value[V any](v *V) V {
	if v == nil {
		var noop V
		return noop
	}
	return *v
}
