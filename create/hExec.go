package create

import (
	"log"
	"os/exec"
)

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
