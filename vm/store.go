package vm

type Store interface {
	Set(module string, source any) error
	Get(module string, dest any) (bool, error)
	Delete(module string)
	Connected() error
	Stop() error
}
