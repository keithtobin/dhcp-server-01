
package main

import (
    "cloud-guy.net/dhcp-server-01/logging"
    "cloud-guy.net/dhcp-server-01/kt"
    "time"
)

func main() {
	l := logging.NewLogger{}
	l.SetAppName("dhcp-server")
	l.Open()
	l.Info("Starting application")
	dhcpServer := kt.DHCPServer{Logger: l}
	dhcpServer.Open()

	//dhcpServer.ProcessIncommingPackets()

	for {
		time.Sleep(time.Second * 1)
	}

	defer dhcpServer.Close()
}

