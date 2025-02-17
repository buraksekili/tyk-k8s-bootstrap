package main

import (
	"fmt"
	"os"
	"tyk/tyk/bootstrap/preinstallation"
)

func main() {
	err := preinstallation.PreHookInstall()
	if err != nil {
		fmt.Printf("Failed to run pre-hook job, err: %v", err)
		os.Exit(1)
	}

	fmt.Println("Pre-Hook bootstrapping succeeded, the provided license is valid!")
}
