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
		fmt.Println("Stopping")
		if err := cmd.Process.Signal(unix.SIGINT); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Waiting")
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
	fswatch := exec.Command("/usr/local/bin/fswatch", ".")
	out, err := fswatch.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("watch: stdout: %v", err)
	}
	if err := fswatch.Start(); err != nil {
		return "", fmt.Errorf("watch: start: %v", err)
	}
	defer func() {
		_ = fswatch.Process.Kill()
		// _, _ = fswatch.Process.Wait()
	}()
	r := bufio.NewReader(out)
	b, err := r.ReadBytes('\n')
	if err != nil {
		return "", fmt.Errorf("watch: read: %v", err)
	}
	p := strings.TrimSpace(string(b))
	return p, nil
}
