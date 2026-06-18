package cache

import "github.com/cgalvisleon/et/config"

var (
	MSG_ATRIB_REQUIRED = "cache, attribute %s is required"
	MSG_UNSUPPORTED_OS = "unsupported os: %s"
)

func init() {
	lang := config.GetStr("LANG", "en")

	if lang == "es" {
		MSG_UNSUPPORTED_OS = "sistema operativo no soportado: %s"
	}
}
