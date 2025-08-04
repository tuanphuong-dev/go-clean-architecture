package common

func New[T any](value T) *T {
	return &value
}
