package inter

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"os"
)

var isterminal bool

func init() {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		isterminal = true
	} else if isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		isterminal = true
	} else {
		isterminal = false
	}
}

func DeletePreviousLine() {
	if isterminal {
		fmt.Print("\u001b[1A\u001b[2K\r")
	}
}
