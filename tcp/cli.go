package tcp

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func StartConsole(s *Client) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("===================================")
	fmt.Println("  TCP Client Console")
	fmt.Println("  Escribe 'help' para comandos")
	fmt.Println("===================================")

	for {
		fmt.Print("> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error leyendo comando:", err)
			continue
		}

		input = strings.TrimSpace(input)
		handleCommand(s, input)
	}
}

func handleCommand(s *Client, cmd string) {
	args := strings.Split(cmd, " ")

	switch args[0] {

	case "help":
		fmt.Println("Comandos disponibles:")
		fmt.Println("  help        - mostrar comandos")
		fmt.Println("  nodes       - listar nodos")
		fmt.Println("  clients     - listar clientes")
		fmt.Println("  leader      - mostrar líder")
		fmt.Println("  stats       - estadísticas")
		fmt.Println("  stop        - detener servidor")

	case "nodes":
		fmt.Println("Nodos:")

	case "clients":
		fmt.Println("Clientes:")
		fmt.Println("  ", s.Addr)
	case "leader":
		fmt.Println("Leader:", s.Addr)

	case "stats":
		fmt.Println("Estadísticas:")
		fmt.Println("  Nodos:", 0)
		fmt.Println("  Clientes:", 1)
		fmt.Println("  Líder:", s.Addr)

	case "stop":
		fmt.Println("Deteniendo cliente...")
		s.Close()
		os.Exit(0)

	default:
		fmt.Println("Comando no reconocido:", cmd)
	}
}
