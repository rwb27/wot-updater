# Web of Things Updater

This utility aims to make it easy to work with an IoT device that's connected to your computer, but not connected to the internet. It does this by *temporarily* providing a proxy through your internet connection. To get started (assuming you have a built executable):

* Run `sshterm -hostname <my_host> -user <my_user>`
* Enter your password when prompted
* Run `export-wot-proxy` once logged in

Everything you run in that session should now have access to the internet via a socks5 proxy on `localhost:10800`.

## The problem

I have a web-of-things device (for example, an [OpenFlexure Microscope]), connected to my computer via a network cable.  This is simple, reliable, and fast, and I can use my microscope just fine.  However, I can't do a software update, because that requires the microscope to connect to the internet.  

I could, of course, connect my microscope to WiFi, but if I don't have a suitable network available (e.g. I'm at work, with complicated Eduroam WiFi, or I'm in a location that requires browser-based authentication, which is a pain from a headless device) I'm stuck.  That leads to me messing about with extra routers, or trying to set up internet connection sharing, all of which are complicated, unreliable, and in many cases very much against the policy of my IT department.

## The solution

I can connect to my microscope using `ssh` and create a "reverse tunnel" back a proxy server on my computer.  This allows temporary use of my computer's internet connection by the microscope, e.g. to download a software update.  On Linux, everything you need to do this is provided by `ssh` and `sshd` with appropriate configuration.  On Windows, however, it doesn't work because the SSH client doesn't have the requisite proxy server feature enabled in the "reverse" direction.

This little Go project bundles up everything I need into a single executable.  When run from the command line, it:

* Makes an SSH connection to the device
* Creates a reverse tunnel (on port 10800)
* Runs a SOCKS5 proxy on my computer, listening to the tunnel
* Sets the date on the remote machine (with `sudo date`)
* Adds bash aliases to set the proxy environment variables

Currently its configuration options are limited, but I expect it will be a really handy utility for updating Web of Things devices on "dark" networks.  This should solve a longstanding and really annoying limitation of such devices in University labs, where (for good reasons) campus IT administrators are generally very unwilling to allow these devices on the main network.  I hope this means that "dark" networks (i.e. not directly connected to either the secure campus network or the internet) become a bit easier to manage.

## Current status

This program currently runs and works on my Windows laptop, connecting to a locally-connected Raspberry Pi running a customised Raspberry Pi OS.  It supports password authentication, accepting the password on the command line, and takes command-line arguments for username, hostname, and port number.  Once connected, several commands are run - configuration for these is very much a to-do I'm afraid. The commands are echoed to the terminal so you can see what it's done. Commands are:

1. Set the date on the Pi (because there's no RTC on a Pi, the date is often wrong and this will cause security warnings): `sudo date -s "%s"` where `%s` is today's date/time on your computer.
2. Create a configuration file for `proxychains` at `~/.proxychains-wot-updater.conf` pointing to localhost:10800.
3. Create several new lines in your `~/.bashrc` file on the Pi using `alias` to define new commands:
  * `export-wot-proxy` sets your proxy environment variables (`http_proxy`, `https_proxy`, `all_proxy` and the same in upper-case) to `socks5h://127.0.0.1:10800`. It also configures `proxychains` to use the configuration file from (2).
  * `export-empty-proxy` sets the proxy environment variables to `""`, disabling the proxy. It doesn't reset the `proxychains` configuration. Typically, if you have a command that doesn't work after `export-wot-proxy` you can use this command to disable the proxy, then try `proxychains <your command>` to use proxychains instead.
  * `sudo-wot-pc` will run a command as root, with no proxy environment variables set, using `proxychains` to redirect network requests through the proxy.
  * `skip-git-lfs` and `dont-skip-git-lfs` allow you to skip the LFS "smudge" filter when using `git`. Because `git-lfs` is written in Go, it won't work with proxychains, and it may not know to send DNS queries via the proxy: this can lead to `git` hanging after it's retrieved the references. If you don't need LFS files, `skip-git-lfs` may be all you need to get it working.

I use `libproxychains4` to connect to the proxy if, for whatever reason, the environment variables are ignored or the program doesn't support it.  With the proxy variables exported (e.g. by typing `export-wot-proxy` in the session) most things work (`pip`, `sudo apt ...`, etc.) but there are edge cases that fail (see below).

### Known issues

* `git-lfs` seems not to use the proxy correctly (I believe this is due to a difference in how the Go http libraries work vs libc). Currently I work around this by exporting `GIT_LFS_SKIP_SMUDGE=1` which is aliased to `disable-git-lfs` by this script.

## Installation and use

Currently this is not packaged/built automatically.  You will need to:
* [Install Go](https://go.dev/doc/install)
* Clone the repository
* Obtain the requisite libraries:
    - `go mod tidy`
* Compile and run the program:
    - `go run ./sshterm`

## Building

You can build a stand-alone, statically linked binary for your platform with `go build ./sshterm`.  Once this is a bit more ready for use, I'll build it automatically in the CI.

[OpenFlexure Microscope]: https://openflexure.org/