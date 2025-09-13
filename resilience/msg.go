package resilience

const (
	MSG_RESILIENCE_NOT_INITIALIZED = "resilience no esta inicializado"
	MSG_ID_REQUIRED                = "id es requerido"
	MSG_ID_NOT_FOUND               = "id no encontrado"
	MSG_INSTANCE_STOPPED           = "intento detenido"
	MSG_INSTANCE_RESTARTED         = "intento reiniciado"
	MSG_INSTANCE_NOT_FOUND         = "intento no encontrado"
	MSG_RESILIENCE_STATUS          = "Attempt:%d de %d Instance:%s, Tag:%s Status:%s"
	MSG_RESILIENCE_ERROR           = "Attempt:%d de %d Error en Instance:%s Tag:%s Status:%s, Error:%s"
	MSG_RESILIENCE_FINISHED        = "Attempt:%d de %d Finalizado Instance:%s Tag:%s Status:%s"
	MSG_RESILIENCE_FINISHED_ERROR  = "Attempt:%d de %d Finalizado con error Instance:%s Tag:%s Status:%s, Error:%s"
)
