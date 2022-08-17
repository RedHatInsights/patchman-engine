package main

import (
	"app/platform"
	"fmt"
	"io/ioutil"
	"os"
)

// when debugging listener, send custom kafka messages
func main() {
	// check args
	if len(os.Args) != 2 {
		fmt.Println("Simple developer script for sending custom kafka messages")
		fmt.Println("Usage: send-kafka-msg <topic> < message_file.json")
		os.Exit(1)
	}

	// parse args
	topic := os.Args[1]

	// read input json
	message, err := ioutil.ReadAll(os.Stdin)

	if err != nil {
		panic(err)
	}

	platform.SendMessageToTopic(topic, string(message))
}
