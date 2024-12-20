package linq

type TypeVar int

// Var field system name
const (
	IdTField TypeVar = iota
	SourceField
	IndexField
	StateField
)

/**
* Up return upcase to field system
* @return string
**/
func (t TypeVar) Up() string {
	switch t {
	case IdTField:
		return "_IDT"
	case SourceField:
		return "_DATA"
	case IndexField:
		return "_INDEX"
	case StateField:
		return "_STATE"
	}

	return ""
}

/**
* Low return lowcase to field system
* @return string
**/
func (t TypeVar) Low() string {
	switch t {
	case IdTField:
		return "_idt"
	case SourceField:
		return "_data"
	case IndexField:
		return "_index"
	case StateField:
		return "_state"
	}

	return ""
}
