package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Request struct {
	Method string `json:"method"`
}

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  string `json:"result"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		if req.Method == "ping" {
			resp := Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  "pong",
			}
			data, _ := json.Marshal(resp)
			fmt.Println(string(data))
		}
	}
}
