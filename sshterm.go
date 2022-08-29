//package build.openflexure.org/wot-updater-ssh
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/things-go/go-socks5"
	"golang.org/x/crypto/ssh"
)

func scanConfig() string {
	config, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	config = strings.Trim(config, "\n")
	return config
}

func scanConfigWithDefault(default_value string) string {
	config := scanConfig()
	if config == "" {
		return default_value
	} else {
		return config
	}
}

func prompt(message string, default_value string) string {
	fmt.Printf("%s (default value: %s)? ", message, default_value)
	return scanConfigWithDefault(default_value)
}

func setRemoteDate(conn *ssh.Client) {
	// Create a session
	date_session, err := conn.NewSession()
	if err != nil {
		log.Fatal("unable to create session: ", err)
	}
	defer date_session.Close()

	// Set the date/time on remote machine
	// Probably, the Raspberry Pi's emulated clock is wrong, which will cause problems.
	date_command := fmt.Sprintf("sudo date -s \"%s\"", time.Now().Format(time.UnixDate))
	fmt.Println("Setting remote time with command:", date_command)
	date_output, err := date_session.CombinedOutput(date_command)
	if err != nil {
		fmt.Printf("Date output: %s", date_output)
		log.Fatal("Unable to set date: ", err)
	}
}

func main() {
	// This is heavily borrowed from the crypto/ssh example
	// with much help from https://github.com/inatus/ssh-client-go/blob/master/main.go
	// Create client config
	config := &ssh.ClientConfig{
		User: "pi",
		Auth: []ssh.AuthMethod{
			ssh.Password("openflexure"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// Connect to ssh server
	conn, err := ssh.Dial("tcp", "microscope.local:22", config)
	if err != nil {
		log.Fatal("unable to connect: ", err)
	}
	defer conn.Close()

	// Create a SOCKS5 server
	socksServer := socks5.NewServer(
		socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
	)
	// Listen on remote host
	listener, err := conn.Listen("tcp", "localhost:10800")
	if err != nil {
		log.Fatal("unable to register reverse port forward: ", err)
	}
	defer listener.Close()
	// Connect the SOCKS5 server to the port on remote host
	go socksServer.Serve(listener)

	setRemoteDate(conn)

	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		log.Fatal("unable to create session: ", err)
	}
	defer session.Close()

	// Set IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
		log.Fatal("request for pseudo terminal failed: ", err)
	}
	// Start remote shell
	if err := session.Shell(); err != nil {
		log.Fatal("failed to start shell: ", err)
	}

	// Wait until the shell finishes (i.e. we log out)
	if err := session.Wait(); err != nil {
		log.Fatal("something went wrong executing the shell: ", err)
	}

	log.Print("Successfully closed the remote connection.")
}
