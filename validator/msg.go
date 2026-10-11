package validator

import "github.com/cgalvisleon/et/envar"

var (
	MSG_VALIDATOR_EMPTY              = "data cannot be empty"
	MSG_VALIDATOR_REQUIRED           = "the field %s is required"
	MSG_VALIDATOR_NOT_EMPTY          = "the field %s cannot be empty"
	MSG_VALIDATOR_MIN                = "the field %s must be greater than %v"
	MSG_VALIDATOR_MAX                = "the field %s must be less than %v"
	MSG_VALIDATOR_BETWEEN            = "the field %s must be between %v and %v"
	MSG_VALIDATOR_MIN_LENGTH         = "the field %s must have at least %v characters"
	MSG_VALIDATOR_MAX_LENGTH         = "the field %s must have less than %v characters"
	MSG_VALIDATOR_PATTERN            = "the field %s does not match the pattern %s"
	MSG_VALIDATOR_LETTERS            = "the field %s must contain at least one letter"
	MSG_VALIDATOR_NUMBERS            = "the field %s must contain at least one number"
	MSG_VALIDATOR_SPECIAL_CHARACTERS = "the field %s must contain at least one special character"
	MSG_VALIDATOR_INVALID_TYPE       = "the field %s is of invalid type"
	MSG_VALIDATOR_EMAIL              = "the field %s must be a valid email"
	MSG_VALIDATOR_PHONE              = "the field %s must be a valid phone"
	MSG_VALIDATOR_UNKNOWN_CONDITION  = "the field %s has an unknown condition %s"
	MSG_VALIDATOR_INVALID_PATTERN    = "the field %s has an invalid pattern %s"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_VALIDATOR_EMPTY = "datos no pueden estar vacíos"
		MSG_VALIDATOR_REQUIRED = "el campo %s es requerido"
		MSG_VALIDATOR_NOT_EMPTY = "el campo %s no puede estar vacío"
		MSG_VALIDATOR_MIN = "el campo %s debe ser mayor a %v"
		MSG_VALIDATOR_MAX = "el campo %s debe ser menor a %v"
		MSG_VALIDATOR_BETWEEN = "el campo %s debe estar entre %v y %v"
		MSG_VALIDATOR_MIN_LENGTH = "el campo %s debe tener al menos %v caracteres"
		MSG_VALIDATOR_MAX_LENGTH = "el campo %s debe tener menos de %v caracteres"
		MSG_VALIDATOR_PATTERN = "el campo %s no cumple con el patrón %s"
		MSG_VALIDATOR_LETTERS = "el campo %s debe contener al menos una letra"
		MSG_VALIDATOR_NUMBERS = "el campo %s debe contener al menos un número"
		MSG_VALIDATOR_SPECIAL_CHARACTERS = "el campo %s debe contener al menos un carácter especial"
		MSG_VALIDATOR_INVALID_TYPE = "el campo %s es de tipo inválido"
		MSG_VALIDATOR_EMAIL = "el campo %s debe ser un correo electrónico válido"
		MSG_VALIDATOR_PHONE = "el campo %s debe ser un teléfono válido"
		MSG_VALIDATOR_UNKNOWN_CONDITION = "el campo %s tiene una condición desconocida %s"
		MSG_VALIDATOR_INVALID_PATTERN = "el campo %s tiene un patrón inválido %s"
	}
}
