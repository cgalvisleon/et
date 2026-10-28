package daemon

import (
	"os"

	"github.com/cgalvisleon/et/et"
)

// TypeCommand: a daemon sub-command (help, version, status, start, stop, restart, conf).
type TypeCommand string

const (
	CMD_Help    TypeCommand = "help"
	CMD_Version TypeCommand = "version"
	CMD_Status  TypeCommand = "status"
	CMD_Start   TypeCommand = "start"
	CMD_Stop    TypeCommand = "stop"
	CMD_Restart TypeCommand = "restart"
	CMD_Conf    TypeCommand = "conf"
)

/**
* ToTypeCommand: The command named by a CLI argument (long or short form); unknown gives CMD_Help.
* @param val string
* @return TypeCommand
**/
func ToTypeCommand(val string) TypeCommand {
	switch val {
	case "version", "--v":
		return CMD_Version
	case "status", "--s":
		return CMD_Status
	case "start", "--start":
		return CMD_Start
	case "stop", "--stop":
		return CMD_Stop
	case "restart", "--restart":
		return CMD_Restart
	case "conf", "--conf":
		return CMD_Conf
	}

	return CMD_Help
}

// RepositoryCMD: what a daemon implementation must offer to the CLI.
type RepositoryCMD interface {
	Help(key string)
	Version() string
	SetConfig(cfg string)
	Status() et.Json
	Start() et.Item
	Stop() et.Item
	Restart() et.Item
}

var apps map[string]RepositoryCMD
var app string = "systemd"

/**
* main: Dispatches os.Args[1] to the registered app.
**/
func main() {
	_app := apps[app]

	if len(os.Args) < 2 {
		_app.Help("")
		return
	}

	command := os.Args[1]
	switch ToTypeCommand(command) {
	case CMD_Version:
		_app.Version()
	case CMD_Help:
		if len(os.Args) > 2 {
			_app.Help(os.Args[2])
		} else {
			_app.Help("")
		}
	case CMD_Conf:
		if len(os.Args) > 2 {
			_app.SetConfig(os.Args[2])
		}
		println("Configuración: Sin parametros")
	case CMD_Status:
		_app.Status()
	case CMD_Start:
		_app.Start()
	case CMD_Stop:
		_app.Stop()
	case CMD_Restart:
		_app.Restart()
	}
}

/**
* Registry: Registers a daemon implementation under name and makes it the current app.
* @param name string, cmd RepositoryCMD
**/
func Registry(name string, cmd RepositoryCMD) {
	if apps == nil {
		apps = make(map[string]RepositoryCMD)
	}

	apps[name] = cmd
	app = name
}
