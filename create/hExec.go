package create

import (
	"log"
	"os/exec"
)

/**
* Command: runs each command in coms and collects its stdout; a failing command exits the process.
* @param coms []string
* @return [][]byte, error
**/
func Command(coms []string) ([][]byte, error) {
	var result [][]byte
	for _, com := range coms {
		out, err := exec.Command(com).Output()
		if err != nil {
			log.Fatal(err)
		}
		result = append(result, out)
	}

	return result, nil
}
