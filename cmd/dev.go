// Binary dev automatically restarts the server when changes are detected.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	// Infinite loop, the user has to send ctrl-c to stop the program.
	for {
		if err := compile(); err != nil {
			log.Fatal(err)
		}
		cmd, err := run()
		if err != nil {
			log.Fatal(err)
		}
		p, err := watch()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("File change: %q\n", p)
		fmt.Println("Restarting the server")
		if err := cmd.Process.Signal(unix.SIGINT); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Waiting for the server to stop")
		st, err := cmd.Process.Wait()
		if err != nil && st.ExitCode() != -1 {
			log.Fatal(err)
		}
	}
}

func compile() error {
	fmt.Println("Compiling")
	cmd := exec.Command("/usr/local/go/bin/go", "build", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func run() (*exec.Cmd, error) {
	fmt.Println("Running")
	cmd := exec.Command("./personal-website")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, cmd.Start()
}

func watch() (string, error) {
	// fswatch writes the absolute paths of changed files.
	// the -1 flag will make it exit after a single event.
	fswatch := exec.Command("/usr/local/bin/fswatch", "-1", "--event", "Updated", ".")
	out, err := fswatch.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("watch: stdout: %v", err)
	}
	if err := fswatch.Start(); err != nil {
		return "", fmt.Errorf("watch: start: %v", err)
	}
	r := bufio.NewReader(out)
	b, err := r.ReadBytes('\n')
	if err != nil {
		return "", fmt.Errorf("watch: read: %v", err)
	}
	p := strings.TrimSpace(string(b))
	if _, err := fswatch.Process.Wait(); err != nil {
		return "", fmt.Errorf("watch: wait: %v", err)
	}
	return p, nil
}
