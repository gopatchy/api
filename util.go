package patchy

import "github.com/gopatchy/path"

func IsCreate[T any](obj *T, prev *T) bool {
	return obj != nil && prev == nil
}

func IsUpdate[T any](obj *T, prev *T) bool {
	return obj != nil && prev != nil
}

func IsDelete[T any](obj *T, prev *T) bool {
	return obj == nil && prev != nil
}

func FieldChanged[T any](obj *T, prev *T, p string) bool {
	v1, err := path.Get(obj, p)
	if err != nil {
		panic(err)
	}

	v2, err := path.Get(prev, p)
	if err != nil {
		panic(err)
	}

	return v1 != v2
}

func P[T any](v T) *T {
	return &v
}
