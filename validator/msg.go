package validator

import "github.com/cgalvisleon/et/envar"

var (
	MSG_VALIDATOR_EMPTY        = "data cannot be empty"
	MSG_VALIDATOR_REQUIRED     = "the field %s is required"
	MSG_VALIDATOR_MIN          = "the field %s must be greater than %s"
	MSG_VALIDATOR_MAX          = "the field %s must be less than %s"
	MSG_VALIDATOR_BETWEEN      = "the field %s must be between %s and %s"
	MSG_VALIDATOR_MIN_LENGTH   = "the field %s must have at least %s characters"
	MSG_VALIDATOR_MAX_LENGTH   = "the field %s must have less than %s characters"
	MSG_VALIDATOR_PATTERN      = "the field %s does not match the pattern %s"
	MSG_VALIDATOR_INVALID_TYPE = "the field %s is of invalid type"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_VALIDATOR_EMPTY = "datos no pueden estar vacíos"
		MSG_VALIDATOR_REQUIRED = "el campo %s es requerido"
		MSG_VALIDATOR_MIN = "el campo %s debe ser mayor a %s"
		MSG_VALIDATOR_MAX = "el campo %s debe ser menor a %s"
		MSG_VALIDATOR_BETWEEN = "el campo %s debe estar entre %s y %s"
		MSG_VALIDATOR_MIN_LENGTH = "el campo %s debe tener al menos %s caracteres"
		MSG_VALIDATOR_MAX_LENGTH = "el campo %s debe tener menos de %s caracteres"
		MSG_VALIDATOR_PATTERN = "el campo %s no cumple con el patrón %s"
		MSG_VALIDATOR_INVALID_TYPE = "el campo %s es de tipo inválido"
	}
}
