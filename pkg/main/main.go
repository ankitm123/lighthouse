package main

import (
	"fmt"
	"os"

	"github.com/jenkins-x/lighthouse/pkg/webhook"
)

// Entrypoint for the command
func main() {
	cmds := webhook.NewCmdWebhook()
	err := cmds.Execute()
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	os.Exit(0)

}