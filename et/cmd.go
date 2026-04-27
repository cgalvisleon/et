package et

type Command string

const (
	INSERT Command = "insert"
	UPDATE Command = "update"
	DELETE Command = "delete"
	UPSERT Command = "upsert"
)

/**
* From
* @param itmes []Json, as string
* @return *Where
**/
func From(itmes []Json, as string) *Where {
	result := newWhere(&Source{
		data: itmes,
		as:   as,
	})
	return result
}
