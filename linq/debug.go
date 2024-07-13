package linq

import "github.com/cgalvisleon/et/logs"

func debug(l *Linq) {
	logs.Debug(l.Definition().ToString())
	logs.Debug(l.Sql)
}
