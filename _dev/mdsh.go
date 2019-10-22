// Takes markdown on stdin and outputs same markdown with shell commands expanded
//
// ```sh (exec)
// $ echo test
// ```
// Becomes:
// ```sh (exec)
// $ echo test
// test
// ```
//
// [echo test]: sh
// Becomes:
// [echo test]: sh
// test
//
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	inExample := false
	// [echo bla]: exec
	execRe := regexp.MustCompile(`\[(.*)\]: sh`)

	for scanner.Scan() {
		l := scanner.Text()

		if inExample {
			if strings.HasPrefix(l, "```") {
				inExample = false
				fmt.Println(l)
			} else {
				fmt.Println(l)
				if strings.HasPrefix(l, "$") {
					cmd := exec.Command("sh", "-c", l[1:])
					o, _ := cmd.CombinedOutput()
					fmt.Print(string(o))
				}
			}
		} else {
			if strings.HasPrefix(l, "```sh (exec)") {
				inExample = true
			}
			fmt.Println(l)

			sm := execRe.FindStringSubmatch(l)
			if sm != nil {
				cmd := exec.Command("sh", "-c", sm[1])
				o, _ := cmd.CombinedOutput()
				fmt.Print(string(o))
			}
		}
	}
}
