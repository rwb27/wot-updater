//package build.openflexure.org/wot-updater-ssh
package main

import (
	"fmt"
	//"net"

	"github.com/hashicorp/mdns"
)

func main() {
	/*// Windows has problems if you don't specify an interface :(
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Print(fmt.Errorf("localAddresses: %v\n", err.Error()))
			continue
		}
		for _, a := range addrs {
			log.Printf("%v %v\n", i.Name, a)
		}
	}*/
	// Discover services using mDNS
	// The channel and function below just print services as they are discovered
	entriesCh := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for entry := range entriesCh {
			fmt.Printf("%s\n", entry.Name)
			fmt.Printf("  AddrV4: %s\n  AddrV6: %s\n", entry.AddrV4, entry.AddrV6)
			fmt.Printf("  Port: %d\n", entry.Port)
			for _, field := range entry.InfoFields {
				fmt.Printf("  Info: %s\n", field)
			}
			fmt.Print("\n")
		}
	}()

	// Start the lookup
	queryParams := mdns.DefaultParams("_labthing._tcp")
	queryParams.Entries = entriesCh
	queryParams.DisableIPv6 = false
	mdns.Query(queryParams)
	close(entriesCh)
}
