package event

import "github.com/cgalvisleon/et/envar"

var (
	MSG_UNSUPPORTED_OS = "unsupported os: %s"
	MSG_PANIC_IN_SUBSCRIBE = "panic in Subscribe channel:%s err:%v"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_UNSUPPORTED_OS = "sistema operativo no soportado: %s"
		MSG_PANIC_IN_SUBSCRIBE = "panic en Subscribe canal:%s err:%v"
	}
}
