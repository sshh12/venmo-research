package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const cmdTemplate = `
set timeout -1
spawn COMMAND
expect {
  "Password:" {send -- "PASSWORD\r" ; exp_continue}
  "TACC Token Code:" {send -- "TOKEN\r" ; exp_continue}
  eof
}`

const tempFn = "login.expect"

func main() {

	var username string
	var password string
	var code string
	var host string
	var sshArgs string
	flag.StringVar(&username, "user", "", "TACC username")
	flag.StringVar(&password, "pw", "", "TACC password")
	flag.StringVar(&code, "code", "", "TACC 2FA Code")
	flag.StringVar(&host, "host", "stampede2.tacc.utexas.edu", "TACC host")
	flag.StringVar(&sshArgs, "ssh_args", "-o \"StrictHostKeyChecking=no\" -f -N -L LOCAL_PORT:HOST:REMOTE_PORT", "ssh arguments")
	localPort := flag.Int("local_port", 8080, "local port")
	remotePort := flag.Int("remote_port", 8888, "remote port")
	flag.Parse()

	if username == "" || password == "" {
		log.Fatal("Invalid login")
		return
	}

	if code == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("2FA Code: ")
		code, _ = reader.ReadString('\n')
	}

	sshCmd := fmt.Sprintf("ssh %s %s@%s", sshArgs, username, host)
	sshCmd = strings.ReplaceAll(sshCmd, "HOST", host)
	sshCmd = strings.ReplaceAll(sshCmd, "LOCAL_PORT", fmt.Sprint(*localPort))
	sshCmd = strings.ReplaceAll(sshCmd, "REMOTE_PORT", fmt.Sprint(*remotePort))

	expectScript := strings.ReplaceAll(cmdTemplate, "PASSWORD", password)
	expectScript = strings.ReplaceAll(expectScript, "TOKEN", code)
	expectScript = strings.ReplaceAll(expectScript, "COMMAND", sshCmd)

	err := ioutil.WriteFile(tempFn, []byte(expectScript), 0600)
	if err != nil {
		log.Fatal(err)
		return
	}

	go func() {
		time.Sleep(2 * time.Second)
		os.Remove(tempFn)
	}()

	cmd := exec.Command("bash", "-c", "expect "+tempFn)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

}
