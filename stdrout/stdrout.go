package stdrout

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/cgalvisleon/et/timezone"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Purple = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"
var useColor = true

func init() {
	if runtime.GOOS == "windows" {
		Reset = ""
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Purple = ""
		Cyan = ""
		Gray = ""
		White = ""
		useColor = false
	}
}

func Printl(kind string, color string, args ...any) string {
	kind = strings.ToUpper(kind)
	message := fmt.Sprint(args...)
	now := timezone.Now()
	var result string

	switch color {
	case "Reset":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + message + Reset
	case "Red":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Red + message + Reset
	case "Green":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Green + message + Reset
	case "Yellow":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Yellow + message + Reset
	case "Blue":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Blue + message + Reset
	case "Purple":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Purple + message + Reset
	case "Cyan":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Cyan + message + Reset
	case "Gray":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Gray + message + Reset
	case "White":
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + White + message + Reset
	default:
		result = now + Purple + fmt.Sprintf(" [%s]: ", kind) + Reset + Green + message + Reset
	}

	println(result)

	return result
}
