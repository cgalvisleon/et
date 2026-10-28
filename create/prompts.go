package create

import (
	"fmt"
	"strconv"

	"github.com/manifoldco/promptui"
)

/**
* PromptCreate: asks what to create (Project, Microservice, Modelo, Rpc) and runs the matching command.
**/
func PromptCreate() {
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
		/* Permite crear un proyecto */
		err := CmdProject.Execute()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
	case 1:
		/* Permite crear un microservicio */
		err := CmdMicro.Execute()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
	case 2:
		/* Permite crear un modelo */
		err := CmdModelo.Execute()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
	case 3:
		/* Permite crear un servicio rpc */
		err := CmdRpc.Execute()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
	}
}

/**
* PromptStr: asks for a text value; with require, an empty answer is rejected.
* @param label string, require bool
* @return string, error
**/
func PromptStr(label string, require bool) (string, error) {
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

/**
* PromptInt: asks for an integer value; non-numeric answers are rejected.
* @param label string, require bool
* @return int, error
**/
func PromptInt(label string, require bool) (int, error) {
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
