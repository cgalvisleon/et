package sql

import "github.com/cgalvisleon/et/et"

type Command string

const (
	INSERT Command = "insert"
	UPDATE Command = "update"
	DELETE Command = "delete"
	UPSERT Command = "upsert"
	FROM   Command = "from"
)

func From(itmes []et.Json) *Where {
	result := newWhere(itmes)
	return result
}
