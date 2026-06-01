package main

import "secrets/internal/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
