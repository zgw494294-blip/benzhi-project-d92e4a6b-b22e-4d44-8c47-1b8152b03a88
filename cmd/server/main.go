package main

import (
	"fmt"
	"os"
)

func main() {
	configuration, err := parseConfig(os.Args[1:])
	if err == nil {
		if configuration.SelfCheck {
			err = runSelfCheck(configuration)
		} else {
			err = runServer(configuration)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}
