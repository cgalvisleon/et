package tcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Console struct {
	client *Client
}

/**
* StartConsole: Starts an interactive CLI for the given client.
* @param client *Client
**/
func StartConsole(client *Client) {
	c := &Console{client: client}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("===================================")
	fmt.Println("  TCP Client Console")
	fmt.Printf("  Addr: %s\n", client.Addr)
	fmt.Println("  Type 'help' for commands")
	fmt.Println("===================================")

	for {
		fmt.Print("> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if c.handleCommand(input) {
			return
		}
	}
}

/**
* handleCommand: Parses and executes a command. Returns true to exit.
* @param line string
* @return bool
**/
func (s *Console) handleCommand(line string) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {

	case "help":
		fmt.Println("Commands:")
		fmt.Println("  help                     - show this help")
		fmt.Println("  status                   - show connection status")
		fmt.Println("  ping                     - ping the server")
		fmt.Println("  request <Method> [args]  - call a server method and print response")
		fmt.Println("  send <payload>           - send a message without waiting for response")
		fmt.Println("  quit / exit              - close and exit")

	case "status":
		fmt.Printf("  Addr:   %s\n", s.client.Addr)
		fmt.Printf("  Status: %s\n", s.client.Status)

	case "ping":
		res := s.client.Request("Tcp.Ping", s.client.ID)
		if res.Error != nil {
			fmt.Println("Error:", res.Error)
		} else {
			printResponse(res)
		}

	case "request":
		if len(args) == 0 {
			fmt.Println("Usage: request <Method.Name> [arg1] [arg2] ...")
			return false
		}
		method := args[0]
		iargs := make([]any, len(args)-1)
		for i, a := range args[1:] {
			iargs[i] = a
		}
		res := s.client.Request(method, iargs...)
		if res.Error != nil {
			fmt.Println("Error:", res.Error)
		} else {
			printResponse(res)
		}

	case "send":
		if len(args) == 0 {
			fmt.Println("Usage: send <payload>")
			return false
		}
		payload := strings.Join(args, " ")
		ms, err := NewMessage(Method, payload)
		if err != nil {
			fmt.Println("Error creating message:", err)
			return false
		}
		if err = s.client.Send(ms); err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("Sent.")
		}

	case "quit", "exit":
		fmt.Println("Closing...")
		s.client.Close()
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %q — type 'help' for available commands\n", cmd)
	}

	return false
}

/**
* printResponse: Prints a Response as formatted JSON.
* @param res *Response
**/
func printResponse(res *Response) {
	bt, err := json.MarshalIndent(res.Response, "", "  ")
	if err != nil {
		fmt.Println(res.Response)
		return
	}
	fmt.Println(string(bt))
}
