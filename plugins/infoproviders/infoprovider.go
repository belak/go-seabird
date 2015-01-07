package infoproviders

type InfoProvider interface {
	Get() interface{}
}
