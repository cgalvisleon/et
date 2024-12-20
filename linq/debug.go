package linq

import "github.com/cgalvisleon/et/logs"

/**
* debug show the sql
* @param l *Linq
**/
func debug(l *Linq) {
	logs.Debug(l.Sql)
}

/**
* showModel show the model
* @param l *Linq
**/
func showModel(l *Linq) {
	if l.showModel {
		logs.Debug(l.Describe().ToString())
	}
}
