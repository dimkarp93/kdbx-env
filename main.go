package main

import "github.com/dimkarp93/kdbx-env/internal/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
