package iterator

type Iterator[K, V any] interface {
	Next() bool
	Current() (K, V)
	Err() error
}
