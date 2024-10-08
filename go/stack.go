package repobuild

type stack[T any] struct {
	Push     func(T)
	Pop      func() T
	Length   func() int
	Elements func() []T
}

func Stack[T any]() stack[T] {
	slice := make([]T, 0)
	return stack[T]{
		Push: func(i T) {
			slice = append(slice, i)
		},
		Pop: func() T {
			res := slice[len(slice)-1]
			slice = slice[:len(slice)-1]
			return res
		},
		Length: func() int {
			return len(slice)
		},
		Elements: func() []T {
			return slice
		},
	}
}
