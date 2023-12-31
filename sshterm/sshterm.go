// package build.openflexure.org/wot-updater-ssh
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/console"
	"github.com/things-go/go-socks5"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
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
	// Set the date/time on remote machine
	// Probably, the Raspberry Pi's emulated clock is wrong, which will cause problems.
	cmd := fmt.Sprintf("sudo date -s \"%s\"", time.Now().Format(time.UnixDate))
	runCommandOnRemoteHost(conn, cmd, "Setting date")
}

func proxy_env_vars(proxy string) string {
	// Return a string that sets all the proxy-related environment variables
	env_var_string :=
		"http_proxy=" + proxy + " " +
			"https_proxy=" + proxy + " " +
			"all_proxy=" + proxy + " " +
			"HTTP_PROXY=" + proxy + " " +
			"HTTPS_PROXY=" + proxy + " " +
			"ALL_PROXY=" + proxy + " "
	return env_var_string
}

func replace_alias_in_bashrc_cmd(alias string, cmd string) string {
	// Return a command that will replace or add a line to .bashrc to define an alias.
	sed_cmd := fmt.Sprintf(
		"sed -i.bak -n -e '/^alias %s/!p' -e '$aalias %s=\"%s\"' ~/.bashrc",
		alias, alias, cmd,
	)
	return sed_cmd
}

func setRemoteAliases(conn *ssh.Client) {
	// Add bash aliases to set the proxy on the remote machine
	proxy := "socks5h://localhost:10800"
	pc_config := "~/.proxychains-wot-updater.conf"
	export_proxy_cmd :=
		"export " + proxy_env_vars(proxy) +
			" PROXYCHAINS_CONF_FILE=" + pc_config +
			" ssh_proxy='\"'\"'ProxyCommand=nc -X 5 -x localhost:10800 %h %p'\"'\"'"
		// The '\"'\"' escapes a ' character, by ending the single-quoted string
		// and appending a double-quoted string containing a single quote.
	clear_proxy_cmd :=
		"export " + proxy_env_vars("") + "ssh_proxy="
	cmds := []string{
		"echo -e \"strict_chain\\nproxy_dns\\n\\n[ProxyList]\\nsocks5 127.0.0.1 10800\" > " + pc_config,
		replace_alias_in_bashrc_cmd("export-wot-proxy", export_proxy_cmd),
		replace_alias_in_bashrc_cmd("export-empty-proxy", clear_proxy_cmd),
		replace_alias_in_bashrc_cmd("wot-pc", proxy_env_vars("")+" proxychains -f "+pc_config),
		replace_alias_in_bashrc_cmd("sudo-wot-pc", proxy_env_vars("")+" sudo proxychains -f "+pc_config),
		replace_alias_in_bashrc_cmd("skip-git-lfs", "export GIT_LFS_SKIP_SMUDGE=1"),
		replace_alias_in_bashrc_cmd("dont-skip-git-lfs", "export GIT_LFS_SKIP_SMUDGE=1"),
	}
	for _, cmd := range cmds {
		runCommandOnRemoteHost(conn, cmd, "Setting aliases")
	}
}

func runCommandOnRemoteHost(conn *ssh.Client, cmd string, description string) ([]byte, error) {
	// Create a session on the remote machine, run a command, and print any errors.
	session, err := conn.NewSession()
	if err != nil {
		log.Fatal("unable to create session: ", err)
	}
	defer session.Close()

	// Run a command on the remote machine (in its own session)
	fmt.Println(description, " on remote machine: `", cmd, "`")
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		fmt.Printf("Command output: %s", output)
		log.Fatal("Command failed: ", err)
	}
	return output, err
}

func askUserSecretQuestions(user, instruction string, questions []string, echos []bool) ([]string, error) {
	// Prompt the user for answers, to support keyboard-interactive authentication
	answers := make([]string, len(questions))
	for i := range answers {
		fmt.Print(questions[i])
		passwd, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return make([]string, 0), err
		}
		answers[i] = string(passwd)
	}

	return answers, nil
}
func askUserForPassword() (string, error) {
	// Prompt the user for a password
	fmt.Print("Password: ")
	passwd, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal("unable to read password: ", err)
	}
	fmt.Println()
	return string(passwd), err
}

func main() {
	// This is heavily borrowed from the crypto/ssh example
	// with much help from https://github.com/inatus/ssh-client-go/blob/master/main.go

	// Parse command line arguments
	hostnamePtr := flag.String("hostname", "microscope.local", "hostname of the microscope")
	portPtr := flag.Int("port", 22, "port number to connect to")
	userPtr := flag.String("user", "pi", "username to log in as")
	killPtr := flag.Bool("kill-listeners", false, "Whether to kill any processes already using port 10800")
	flag.Parse()

	// Create client config
	config := &ssh.ClientConfig{
		User: *userPtr,
		Auth: []ssh.AuthMethod{
			//ssh.Password("openflexure"),
			ssh.PasswordCallback(askUserForPassword),
			ssh.KeyboardInteractive(askUserSecretQuestions),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// Connect to ssh server
	conn, err := ssh.Dial("tcp", net.JoinHostPort(*hostnamePtr, fmt.Sprint(*portPtr)), config)
	if err != nil {
		log.Fatal("unable to connect: ", err)
	}
	defer conn.Close()

	if *killPtr {
		runCommandOnRemoteHost(conn, "sudo kill `sudo netstat -nlp | grep 10800 | sed -E 's/.* ([0-9]+).sshd.*/\\1/'`", "Killing processes using port 10800")
	}

	// Create a SOCKS5 server
	socksServer := socks5.NewServer(
		socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
	)
	// Listen on remote host, using both ipv4 and 6, if possible
	listening := false
	protocols := []string{"tcp", "tcp6"}
	for _, p := range protocols {
		listener, err := conn.Listen(p, "localhost:10800")
		if err != nil {
			fmt.Println("unable to register reverse port forward: ", err, " on protocol ", p)
		} else {
			fmt.Println("listening on protocol ", p)
			defer listener.Close()
			// Connect the SOCKS5 server to the port on remote host
			go socksServer.Serve(listener)
			listening = true
		}
	}
	if !listening {
		log.Fatal("Could not listen on port 10800, try using --kill-listeners to recover from a disconnect")
	}

	// Set the date, and add aliases on the remote machine to set proxy environment variables
	setRemoteDate(conn)
	setRemoteAliases(conn)

	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		log.Fatal("unable to create session: ", err)
	}
	defer session.Close()

	// Make the terminal raw, so it works properly with SSH
	// NB doing this with golang.org/x/term doensn't work with
	// arrow keys on Windows.
	current := console.Current()
	defer current.Reset()

	if err := current.SetRaw(); err != nil {
		log.Fatal("couldn't make the terminal raw: ", err)
	}

	// Set IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	colorterm := os.Getenv("COLORTERM")
	terminalType := ""
	if len(colorterm) > 0 {
		fmt.Printf("COLORTERM environment variable: '%s', enabling color output\n", os.Getenv("COLORTERM"))
		terminalType = "xterm"
	}
	// Request pseudo terminal
	if err := session.RequestPty(terminalType, 40, 80, modes); err != nil {
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
