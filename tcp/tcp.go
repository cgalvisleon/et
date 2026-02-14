package tcp

/**
* Ping
* @param to *Client
* @return string, error
**/
func Ping(to *Client) (string, error) {
	m, err := NewMessage(PingMessage, "")
	if err != nil {
		return "", err
	}

	msg, err := to.request(m)
	if err != nil {
		return "", err
	}

	var response string
	err = msg.Get(&response)
	if err != nil {
		return "", err
	}

	return response, nil
}
