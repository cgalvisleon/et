package instances

type Store interface {
	Set(id, tag string, obj any) error
	Get(id string, dest any) (bool, error)
	Delete(id string) error
}
