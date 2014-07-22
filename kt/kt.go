package kt

import (
	"cloud-guy.net/dhcp-server-01/logging"
	"net"
	"errors"
	"time"
	"strconv"
	"fmt"
)

type DHCPServer struct {
	Logger logging.NewLogger
	Name string
	IP string
	PortI int
	PortO int
	
	isOpen bool
	conn net.PacketConn
	exit bool
}


func (dhcpServer *DHCPServer) Open() error {

	dhcpServer.Logger.Debug("Open() called")

	if dhcpServer.isOpen == true {
		s := "DHCPServer is already open, try closing first"
		dhcpServer.Logger.Error(s)
		return errors.New(s)
	}

	
	if err := dhcpServer.startListing(); err != nil {
		dhcpServer.Logger.Error(err.Error())
		return err
	}

	dhcpServer.isOpen = true

	go dhcpServer.processIncommingPackets()

	dhcpServer.Logger.Debug("Open() OK")

	return nil
}


func (dhcpServer *DHCPServer) Close() error {

	dhcpServer.Logger.Debug("Close() called")

	if dhcpServer.isOpen == false {
		s := "DHCPServer is already closed, try opening first"
		dhcpServer.Logger.Error(s)
		return errors.New(s)
	}

	//exit 
	//wait

	dhcpServer.stopListing()

	dhcpServer.isOpen = false

	dhcpServer.Logger.Debug("Close() OK")

	return nil
}

func (dhcpServer *DHCPServer) startListing() error {

	dhcpServer.Logger.Debug("startListing() called")

	l, err := net.ListenPacket("udp4", ":67")
	if err != nil {
		dhcpServer.Logger.Error(err.Error())
		return err
	}
	dhcpServer.conn = l

	dhcpServer.Logger.Debug("startListing() OK")

	return nil 
}

func (dhcpServer *DHCPServer) stopListing()  {

	dhcpServer.conn.Close()

}

//This function will wait for a UDP packet to arive and
//preform some checks to make sure it is a DHCP packet,
//it will then pass the received packet to processDHCPPacket.
//When processDHCPPacket as finished processing it will return a responce packet
//and this responce packet will be broadcasted. 
func (dhcpServer *DHCPServer) processIncommingPackets() error {

	buffer := make([]byte, 1500)

	for {
		n, addr, err := dhcpServer.conn.ReadFrom(buffer)
		if err != nil {
			return err
		}

		if n < 240 { // Packet too small to be DHCP 
			continue
		}
		
		req := Packet(buffer[:n])
		if req.HLen() > 16 { // Invalid size
			continue
		}

		options := req.ParseOptions()
		var reqType MessageType
		if t := options[OptionDHCPMessageType]; len(t) != 1 {
			continue
		} else {
			reqType = MessageType(t[0])
			if reqType < Discover || reqType > Inform {
				continue
			}
		}

		res, err := dhcpServer.processDHCPPacket(req, reqType, options);
		if  res == nil || err != nil {
			return err
		}

		ipStr, portStr, err := net.SplitHostPort(addr.String())
			if err != nil {
				return err
			}

		if net.ParseIP(ipStr).Equal(net.IPv4zero) || req.Broadcast() {
			port, _ := strconv.Atoi(portStr)
			addr = &net.UDPAddr{IP: net.ParseIP("192.168.182.255"), Port: port}
		}

		if _, e := dhcpServer.conn.WriteTo(res, addr); e != nil {
				return e
		}
	
	}

}

//This function will inspect the incomming UDP packet as received by ProcessIncommingPackets.
//It will then select a function to process the UDP packet.
func (dhcpServer *DHCPServer) processDHCPPacket(p Packet, msgType MessageType, options Options)  (d Packet, e error) {

	switch msgType {

		case Discover:
			return dhcpServer.processDHCPDiscover(p, msgType, options)

		case Request:
			return dhcpServer.processDHCPRequest(p, msgType, options)
		
		case Release, Decline:
			return dhcpServer.processDHCPRelease(p, msgType, options)		
	}

	return nil, nil
} 

//Process the incomming discovery message and create a responce packet.
func (dhcpServer *DHCPServer) processDHCPDiscover(p Packet, msgType MessageType, options Options)  (d Packet, e error) {

	dhcpServer.Logger.Debug("Received a DHCP discovery packet. " + dhcpServer.packetToString(p))


	//free, nic := -1, p.CHAddr().String()

	serverIP := net.IP{192, 168, 182, 1}
	freeIP := net.IP{192, 168, 182, 50}
	leaseDuration := 2 * time.Hour

	staticOptions := Options{
			OptionSubnetMask:       []byte{255, 255, 255, 0},
			OptionRouter:           []byte(serverIP), // Presuming Server is also your router
			OptionDomainNameServer: []byte(serverIP), // Presuming Server is also your DNS server
		}

	retPacket := ReplyPacket(p, Offer, serverIP, freeIP, leaseDuration, staticOptions.SelectOrderOrAll(options[OptionParameterRequestList]))
	dhcpServer.Logger.Debug("Sending responce packet to a DHCP discovery. " + dhcpServer.packetToString(retPacket))

	return retPacket, nil
}

//Process the incomming request message and create a responce packet.
func (dhcpServer *DHCPServer) processDHCPRequest(p Packet, msgType MessageType, options Options)  (d Packet, e error) {

	dhcpServer.Logger.Debug("Received a DHCP request packet. " + dhcpServer.packetToString(p))


	serverIP := net.IP{192, 168, 182, 1}
	leaseDuration := 2 * time.Hour

	staticOptions := Options{
			OptionSubnetMask:       []byte{255, 255, 255, 0},
			OptionRouter:           []byte(serverIP), // Presuming Server is also your router
			OptionDomainNameServer: []byte(serverIP), // Presuming Server is also your DNS server
		}


	if server, ok := options[OptionServerIdentifier]; ok && !net.IP(server).Equal(serverIP) {
			return nil, nil // Message not for this dhcp server
		}

	if reqIP := net.IP(options[OptionRequestedIPAddress]); len(reqIP) == 4 {
				 
	retPacket := ReplyPacket(p, ACK, serverIP, net.IP(options[OptionRequestedIPAddress]), leaseDuration,staticOptions.SelectOrderOrAll(options[OptionParameterRequestList]))

		
	retPacket.SetFile([]byte("keith.txt"))
	retPacket.SetSIAddr([]byte(net.IP{192, 168, 182, 100}))
	
	dhcpServer.Logger.Debug("Sending responce packet to a DHCP request. " + dhcpServer.packetToString(retPacket))

	return retPacket, nil

	}
					
		
	retPacket := ReplyPacket(p, NAK, serverIP, nil, 0, nil)
	dhcpServer.Logger.Debug("Sending responce packet to a DHCP request. " + dhcpServer.packetToString(retPacket))

	return retPacket, nil
}

//Process the incomming release message and create a responce packet.
func (dhcpServer *DHCPServer) processDHCPRelease(p Packet, msgType MessageType, options Options)  (d Packet, e error) {

	dhcpServer.Logger.Debug("Received a DHCP release packet. " + dhcpServer.packetToString(p))


	return nil, nil
}

//This function will take a packet and parse all the values into a formatted readable string 
func (dhcpServer *DHCPServer) packetToString(p Packet) (s string) {

	var retString string

	retString = retString + "OpCode : " + dhcpServer.opCodeToString(p.OpCode())
	retString = retString + " "
	retString = retString + "HType=" + dhcpServer.byteToHexString(p.HType(), ",")
	retString = retString + " "
	retString = retString + "HLen=" + dhcpServer.byteToHexString(p.HLen(),",")
	retString = retString + " "
	retString = retString + "Hops=: " + dhcpServer.byteToHexString(p.Hops(),",")
	retString = retString + " "
	retString = retString + "XId=" + dhcpServer.byteArrayToHexString(p.XId(),"")
	retString = retString + " "
	retString = retString + "Secs=" + dhcpServer.byteArrayToHexString(p.Secs(),"")
	retString = retString + " "
	retString = retString + "Flags=" + dhcpServer.byteArrayToHexString(p.Flags(),",")
	retString = retString + " "
	retString = retString + "CIAddr=" + dhcpServer.byteArrayToDecString(p.CIAddr(),".")
	retString = retString + " "
	retString = retString + "YIAddr=" + dhcpServer.byteArrayToDecString(p.YIAddr(),".")
	retString = retString + " "
	retString = retString + "SIAddr=" + dhcpServer.byteArrayToDecString(p.SIAddr(),".")
	retString = retString + " "
	retString = retString + "GIAddr=" + dhcpServer.byteArrayToDecString(p.GIAddr(),".")
	retString = retString + " "
	retString = retString + "CHAddr=" + p.CHAddr().String()
	retString = retString + " "
	retString = retString + "File=" + string(p.File())
	//SName
	//File
	//Cookie
	//Options
	//Broadcast
	return retString
}


//This function will take a byte array and return a dec string
func (dhcpServer *DHCPServer) opCodeToString(o OpCode)  (s string) {

	if byte(o) == 1 {
		return "BOOTREQUEST (1)"	
	} else if byte(o) == 2 {
		return "BOOTREPLY (2)"
	} else {
		return "UNKNOWN (" + fmt.Sprintf("%d",byte(o)) + ")"
	}

}


//This function will take a byte array and return a dec string
func (dhcpServer *DHCPServer) byteArrayToDecString(b []byte, seperator string)  (s string) {

	var retString string

	for _,i := range b {
		retString = retString + fmt.Sprintf("%d" + seperator, i)
	}

	return retString
}

//This function will take a byte array and return a hex string
func (dhcpServer *DHCPServer) byteArrayToHexString(b []byte, seperator string)  (s string) {

	var retString string

	for _,i := range b {
		retString = retString + fmt.Sprintf("%X" + seperator, i)
	}

	return retString
}

//This function will take a byte  and return a hex string
func (dhcpServer *DHCPServer) byteToHexString(b byte, seperator string)  (s string) {

	var retString string

	retString = retString + fmt.Sprintf("%X", b)

	return retString
}






