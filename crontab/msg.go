package crontab

import "github.com/cgalvisleon/et/config"

var (
	MSG_ADD_JOB          = "Add job:%s tag:%s channel:%s type:%s spec:%s"
	MSG_REMOVE_JOB       = "Remove job:%s"
	MSG_JOB_STORE_IS_NIL = "job store is nil"
	MSG_JOB_STATUS       = "job:%s | status:%s | host:%s | attempt:%d | repetitions:%d"
)

func init() {
	lang := config.GetStr("LANG", "en")

	if lang == "es" {
		MSG_ADD_JOB = "Agregar job:%s tag:%s channel:%s type:%s spec:%s"
		MSG_REMOVE_JOB = "Eliminar job:%s"
		MSG_JOB_STORE_IS_NIL = "job store es nulo"
		MSG_JOB_STATUS = "job:%s | status:%s | host:%s | attempt:%d | repetitions:%d"
	}
}
