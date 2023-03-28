# Web of Things Updater

## The problem

I have a web-of-things device (for example, an [OpenFlexure Microscope]), connected to my computer via a network cable.  This is simple, reliable, and fast, and I can use my microscope just fine.  However, I can't do a software update, because that requires the microscope to connect to the internet.  

I could, of course, connect my microscope to my home WiFi, but if I don't have a suitable network available (e.g. I'm at work, with complicated Eduroam WiFi, or I'm in a location that requires browser-based authentication, which is a pain from a headless device) I'm stuck.  That leads to me messing about with extra routers, or trying to set up internet connection sharing, all of which are complicated, unreliable, and in many cases very much against the policy of my IT department.

## The solution

I can connect to my microscope using `ssh` and create a "reverse tunnel" back a proxy server on my computer.  This allows temporary use of my computer's internet connection by the microscope, e.g. to download a software update.  On Linux, everything you need to do this is provided by `ssh` and `sshd` with appropriate configuration.  On Windows, however, it doesn't work because the SSH client doesn't have the requisite proxy server feature enabled in the "reverse" direction.

This little Go project bundles up everything I need into a single executable.  When run from the command line, it:

* Makes an SSH connection to the device
* Creates a reverse tunnel (on port 10800)
* Runs a SOCKS5 proxy connected to the tunnel
* Sets the date on the remote machine (with `sudo date`)
* Adds bash aliases to set the proxy environment variables

Currently its configuration options are very limited, but I expect it will be a really handy utility for updating Web of Things devices on "dark" networks.  This should solve a longstanding and really annoying limitation of such devices in University labs, where (for good reasons) campus IT administrators are generally very unwilling to allow these devices on the main network.  I hope this means that "dark" networks (i.e. not directly connected to either the secure campus network or the internet) become a bit easier to manage.

## Current status

This program currently runs and works on my Windows laptop, connecting to a locally-connected Raspberry Pi running a customised Raspberry Pi OS.  I use `libproxychains4` to connect to the proxy if, for whatever reason, the environment variables are ignored or the program doesn't support it.  With the proxy variables exported (e.g. by typing `export-wot-proxy` in the session) most things work (`pip`, `sudo apt ...`, etc.) but there are edge cases that fail (see below).

### Known issues

* `git-lfs` seems not to use the proxy correctly (I believe this is due to a difference in how the Go http libraries work vs libc). Currently I work around this by exporting `GIT_LFS_SKIP_SMUDGE=1` which is aliased to `disable-git-lfs` by this script.

### Things that really need to happen before it's useful:

* Allow changing of hard-coded user/password/host/port settings
* Handle command-line arguments (e.g. user, host, port)
* Prompt for a password
* Add documentation

## Installation and use

Currently this is not packaged/built automatically.  You will need to:
* [Install Go](https://go.dev/doc/install)
* Clone the repository
* Obtain the requisite libraries:
    - `go get github.com/things-go/go-socks5`
	- `go get golang.org/x/crypto/ssh`
* Compile and run the program:
    - `go run sshterm.go`

## Building

You can build a stand-alone, statically linked binary for your platform with `go build sshterm.go`.  Once this is a bit more ready for use, I'll build it automatically in the CI.

[OpenFlexure Microscope]: https://openflexure.org/