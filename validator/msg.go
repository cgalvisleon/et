package validator

import "github.com/cgalvisleon/et/envar"

var (
	MSG_VALIDATOR_REQUIRED     = "El campo %s es requerido"
	MSG_VALIDATOR_MIN          = "El campo %s debe ser mayor a %s"
	MSG_VALIDATOR_MAX          = "El campo %s debe ser menor a %s"
	MSG_VALIDATOR_BETWEEN      = "El campo %s debe estar entre %s y %s"
	MSG_VALIDATOR_MIN_LENGTH   = "El campo %s debe tener al menos %s caracteres"
	MSG_VALIDATOR_MAX_LENGTH   = "El campo %s debe tener menos de %s caracteres"
	MSG_VALIDATOR_PATTERN      = "El campo %s no cumple con el patrón %s"
	MSG_VALIDATOR_INVALID_TYPE = "El campo %s es de tipo inválido"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_VALIDATOR_REQUIRED = "El campo %s es requerido"
		MSG_VALIDATOR_MIN = "El campo %s debe ser mayor a %s"
		MSG_VALIDATOR_MAX = "El campo %s debe ser menor a %s"
		MSG_VALIDATOR_BETWEEN = "El campo %s debe estar entre %s y %s"
		MSG_VALIDATOR_MIN_LENGTH = "El campo %s debe tener al menos %s caracteres"
		MSG_VALIDATOR_MAX_LENGTH = "El campo %s debe tener menos de %s caracteres"
		MSG_VALIDATOR_PATTERN = "El campo %s no cumple con el patrón %s"
		MSG_VALIDATOR_INVALID_TYPE = "El campo %s es de tipo inválido"
	}
}
