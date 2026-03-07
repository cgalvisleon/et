package instances

type Store interface {
	Get(id string, dest any) (bool, error)
	Set(id, tag string, obj any) error
	Delete(id string) error
}
