package main

import (
	"github.com/dimkarp93/install-libs/buildinfo"
	"github.com/dimkarp93/kdbx-env/internal/cmd"
)

var (
	version  string
	origin   string
	upstream string
	commit   string
	channel  string
)

func main() {
	cmd.Execute(buildinfo.Info{
		Version:  version,
		Origin:   origin,
		Upstream: upstream,
		Commit:   commit,
		Channel:  channel,
	})
}
