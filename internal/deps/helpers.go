package deps

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// out is where install/update output and guides are written. The GUI service
// points this at a streaming writer; CLI usage keeps os.Stdout.
var out io.Writer = os.Stdout

// SetOutput redirects all informational output from the deps package.
func SetOutput(w io.Writer) {
	if w == nil {
		out = io.Discard
		return
	}
	out = w
}

func Output() io.Writer { return out }

func printf(format string, args ...any) {
	fmt.Fprintf(out, format, args...)
}

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = out
	c.Stderr = out
	return c.Run()
}
