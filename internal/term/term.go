package term

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

var stdinReader = bufio.NewReader(os.Stdin)

func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func ReadWithPrefill(prompt, prefill string) string {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Printf("%s [%s]: ", prompt, prefill)
		line, _ := stdinReader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			return prefill
		}
		return line
	}
	defer term.Restore(fd, oldState)

	fmt.Printf("%s: %s", prompt, prefill)
	buf := []rune(prefill)

	one := make([]byte, 1)
	for {
		n, _ := os.Stdin.Read(one)
		if n == 0 {
			break
		}
		b := one[0]
		switch {
		case b == '\r' || b == '\n':
			fmt.Print("\r\n")
			return string(buf)
		case b == 3 || b == 27:
			fmt.Print("\r\n")
			term.Restore(fd, oldState)
			os.Exit(1)
		case b == 127 || b == 8:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				fmt.Print("\b \b")
			}
		default:
			if b >= 32 {
				buf = append(buf, rune(b))
				os.Stdout.Write([]byte{b})
			}
		}
	}
	return string(buf)
}

func openTTY() *os.File {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil
	}
	return f
}

func confirmYN(question string) bool {
	tty := openTTY()
	if tty == nil {
		fmt.Print(question + " [y/N]: ")
		line, _ := stdinReader.ReadString('\n')
		return strings.EqualFold(strings.TrimSpace(line), "y")
	}
	defer tty.Close()
	fmt.Fprint(tty, question+" [y/N]: ")
	reader := bufio.NewReader(tty)
	line, _ := reader.ReadString('\n')
	return strings.EqualFold(strings.TrimSpace(line), "y")
}

func Confirm(question string, assumeYes bool) bool {
	if assumeYes {
		return true
	}
	return confirmYN(question)
}

func ReadPassword(prompt string) string {
	if v := os.Getenv("SECRETS_PASSWORD"); v != "" {
		return v
	}
	tty := openTTY()
	if tty == nil {
		fmt.Print(prompt)
		line, _ := stdinReader.ReadString('\n')
		return strings.TrimRight(line, "\r\n")
	}
	defer tty.Close()

	fd := int(tty.Fd())
	fmt.Fprint(tty, prompt)
	pw, err := term.ReadPassword(fd)
	if err != nil {
		reader := bufio.NewReader(tty)
		line, _ := reader.ReadString('\n')
		return strings.TrimRight(line, "\r\n")
	}
	fmt.Fprintln(tty)
	return string(pw)
}
