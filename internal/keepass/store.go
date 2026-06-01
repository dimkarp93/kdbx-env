package keepass

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CreateStore(path, password string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	out, err := Run(password+"\n"+password+"\n", "db-create", "-q", "-p", path)
	if err != nil {
		return fmt.Errorf("db-create %s: %w: %s", path, err, strings.TrimSpace(out))
	}
	return nil
}

func AddEmptySecret(path, password, title string) error {
	parts := strings.Split(title, "/")
	for i := 1; i < len(parts); i++ {
		group := strings.Join(parts[:i], "/")
		Run(password+"\n", "mkdir", "-q", path, group)
	}
	out, err := Run(password+"\n", "add", "-q", path, title)
	if err != nil {
		return fmt.Errorf("add %q to %s: %w: %s", title, path, err, strings.TrimSpace(out))
	}
	return nil
}
