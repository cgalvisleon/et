package jfdb

var MSGS map[string]string = map[string]string{}

func init() {
	MSGS["MSG_DATABASE_EXISTS"] = "Database exists"
}

func T(msg string) string {
	return MSGS[msg]
}
