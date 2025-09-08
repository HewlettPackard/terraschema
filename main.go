// (C) Copyright 2024 Hewlett Packard Enterprise Development LP
package main

import (
	"fmt"
	"os"

	"github.com/HewlettPackard/terraschema/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
