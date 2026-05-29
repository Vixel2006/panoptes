package port

type IDGenerator interface {
	New() string
}
