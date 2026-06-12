package stores

import "github.com/cgalvisleon/et/config"

var (
	MSG_RECORD_EXISTS = "record already exists"
)

func init() {
	lang := config.GetStr("LANG", "en")

	if lang == "es" {
		MSG_RECORD_EXISTS = "registro ya existe"
	}
}
