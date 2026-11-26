package types

type ResponseStatusProvider interface {
	WrittenStatus() int
}
