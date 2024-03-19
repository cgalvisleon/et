package create

import (
	"fmt"
	"strconv"

	"github.com/manifoldco/promptui"
)

func PrompCreate() {
	prompt := promptui.Select{
		Label: "What do you want created?",
		Items: []string{"Project", "Microservice", "Modelo", "Rpc"},
	}

	opt, _, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	switch opt {
	case 0:
		// Permite crear un proyecto
		err := CmdProject.Execute()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
	case 1:
		// Permite crear un microservicio
		err := CmdMicro.Execute()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
	case 2:
		// Permite crear un modelo
		err := CmdModelo.Execute()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
	case 3:
		// Permite crear un servicio rpc
		err := CmdRpc.Execute()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
	}
}

func PrompStr(label string, require bool) (string, error) {
	validate := func(input string) error {
		if len(input) == 0 && require {
			return fmt.Errorf("invalid %s", label)
		}

		return nil
	}

	prompt := promptui.Prompt{
		Label:    label,
		Validate: validate,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func PrompInt(label string, require bool) (int, error) {
	validate := func(input string) error {
		if len(input) == 0 && require {
			return fmt.Errorf("invalid %s", label)
		}

		_, err := strconv.Atoi(input)
		if err != nil {
			return fmt.Errorf("invalid %s", label)
		}

		return nil
	}

	prompt := promptui.Prompt{
		Label:    label,
		Validate: validate,
	}

	value, err := prompt.Run()

	if err != nil {
		return 0, err
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return result, nil
}
