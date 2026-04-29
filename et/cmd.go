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
* @param itmes []Json, as ...string
* @return *Where
**/
func From(itmes []Json, as ...string) *Where {
	asStr := "A"
	if len(as) == 1 {
		asStr = as[0]
	}

	return newWhere(&Source{
		data: itmes,
		as:   asStr,
	})
}
