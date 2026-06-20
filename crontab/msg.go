package crontab

import "github.com/cgalvisleon/et/config"

var (
	MSG_ADD_JOB                    = "Add job:%s tag:%s channel:%s type:%s spec:%s"
	MSG_REMOVE_JOB                 = "Remove job:%s"
	MSG_JOB_STORE_IS_NIL           = "job store is nil"
	MSG_JOB_STATUS                 = "job:%s | status:%s | host:%s | attempt:%d | repetitions:%d"
	MSG_ERROR_EXECUTING_JOB        = "error executing job: %s:%s; %s"
	MSG_ERROR_UNMARSHALLING_JOB    = "eventSet error unmarshalling job: %s"
	MSG_ERROR_ADDING_JOB           = "eventSet error adding job: %s"
	MSG_ERROR_STOPPING_JOB         = "eventStop error stopping job: %s"
	MSG_ERROR_STARTING_JOB         = "eventStart error starting job: %s"
	MSG_ERROR_DAY_OF_WEEK_INVALID  = "day of week is invalid"
	MSG_ERROR_MONTH_INVALID        = "month is invalid"
	MSG_ERROR_DAY_OF_MONTH_INVALID = "day of month is invalid"
	MSG_ERROR_HOUR_INVALID         = "hour is invalid"
	MSG_ERROR_MINUTE_INVALID       = "minute is invalid"
	MSG_SEND_JOB_REMOVED           = "send job to be removed"
	MSG_SEND_JOB_STOPPED           = "send job to be stopped"
	MSG_SEND_JOB_STARTED           = "send job to be started"
)

func init() {
	lang := config.GetStr("LANG", "en")

	if lang == "es" {
		MSG_ADD_JOB = "Agregar job:%s tag:%s channel:%s type:%s spec:%s"
		MSG_REMOVE_JOB = "Eliminar job:%s"
		MSG_JOB_STORE_IS_NIL = "job store es nulo"
		MSG_JOB_STATUS = "job:%s | status:%s | host:%s | attempt:%d | repetitions:%d"
		MSG_ERROR_EXECUTING_JOB = "error ejecutando job: %s:%s; %s"
		MSG_ERROR_UNMARSHALLING_JOB = "eventSet error unmarshalling job: %s"
		MSG_ERROR_ADDING_JOB = "eventSet error adding job: %s"
		MSG_ERROR_STOPPING_JOB = "eventStop error stopping job: %s"
		MSG_ERROR_STARTING_JOB = "eventStart error starting job: %s"
		MSG_ERROR_DAY_OF_WEEK_INVALID = "day of week is invalid"
		MSG_ERROR_MONTH_INVALID = "month is invalid"
		MSG_ERROR_DAY_OF_MONTH_INVALID = "day of month is invalid"
		MSG_ERROR_HOUR_INVALID = "hour is invalid"
		MSG_ERROR_MINUTE_INVALID = "minute is invalid"
		MSG_SEND_JOB_REMOVED = "enviar job para eliminar"
		MSG_SEND_JOB_STOPPED = "enviar job para detener"
		MSG_SEND_JOB_STARTED = "enviar job para iniciar"
	}
}
