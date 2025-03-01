package utility

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/cgalvisleon/et/logs"
)

func GetPidByPort(port int) int {
	oss := runtime.GOOS

	findWindowsPid := func(port int) int {
		cmd := exec.Command("cmd", "/C", "netstat -ano | findstr :"+strconv.Itoa(port))
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			return 0
		}

		output := strings.TrimSpace(out.String())
		if output == "" {
			return 0
		}

		re := regexp.MustCompile(`\s+(\d+)$`)
		matches := re.FindStringSubmatch(output)
		if len(matches) < 2 {
			return 0
		}

		pid, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0
		}

		return pid
	}

	findLinuxPid := func(port int) int {
		cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-t")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			return 0
		}

		output := strings.TrimSpace(out.String())
		if output == "" {
			return 0
		}

		pid, err := strconv.Atoi(output)
		if err != nil {
			return 0
		}

		return pid
	}

	switch oss {
	case "windows":
		return findWindowsPid(port)
	case "darwin":
		return findLinuxPid(port)
	case "linux":
		return findLinuxPid(port)
	default:
		logs.Logf("Log", "Sistema operativo desconocido: %s\n", oss)
	}

	return 0
}
