package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// // get the local ip and port based on our destination ip
// func localIPPort(dstip net.IP) (net.IP, int) {
// 	serverAddr, err := net.ResolveUDPAddr("udp", dstip.String()+":12345")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// We don't actually connect to anything, but we can determine
// 	// based on our destination ip what source ip we should use.
// 	if con, err := net.DialUDP("udp", nil, serverAddr); err == nil {
// 		if udpaddr, ok := con.LocalAddr().(*net.UDPAddr); ok {
// 			return udpaddr.IP, udpaddr.Port
// 		}
// 	}
// 	log.Fatal("could not get local ip: " + err.Error())
// 	return nil, -1
// }

func fakeSourceIP(srcip string) net.IP {
	ip := net.ParseIP(srcip)
	if ip == nil {
		log.Fatal("Cannot parse %s", srcip)
	}
	return ip
}

const srcFakeIP = "93.184.216.34"

func main() {
	destIn := flag.String("dest", "", "destination host/ip")
	sportIn := flag.String("sport", "", "source IP")
	dportIn := flag.String("dport", "", "destination IP")
	flag.Parse()
	if *destIn == "" || *dportIn == "" {
		log.Printf("Usage: %s -dest <dest host/ip>  -dport <dport> [-sport <sport>]\n", os.Args[0])
		os.Exit(-1)
	}
	log.Println("starting")

	dstaddrs, err := net.LookupIP(*destIn)
	if err != nil {
		log.Fatal(err)
	}

	// parse the destination host and port from the command line os.Args
	dstip := dstaddrs[0].To4()
	var dstport layers.TCPPort
	if d, err := strconv.ParseUint(*dportIn, 10, 16); err != nil {
		log.Fatal(err)
	} else {
		dstport = layers.TCPPort(d)
	}

	// srcip, sport := localIPPort(dstip)
	srcip := fakeSourceIP(srcFakeIP)

	var sport int
	if *sportIn != "" {
		if tmp, err := strconv.ParseUint(*sportIn, 10, 16); err != nil {
			log.Fatalf("Failed to parse sport %s: err", *sportIn, err)
		} else {
			sport = int(tmp)
		}
	}
	srcport := layers.TCPPort(sport)
	log.Printf("using srcip: %s, sport %d, dstip %s, dport %d", srcip.String(), srcport, dstip.String(), dstport)

	// Our IP header... not used, but necessary for TCP checksumming.
	ip := &layers.IPv4{
		SrcIP:    srcip,
		DstIP:    dstip,
		Protocol: layers.IPProtocolTCP,
	}
	// Our TCP header
	tcp := &layers.TCP{
		SrcPort: srcport,
		DstPort: dstport,
		Seq:     1105024978,
		SYN:     true,
		Window:  14600,
	}
	tcp.SetNetworkLayerForChecksum(ip)

	// Serialize.  Note:  we only serialize the TCP layer, because the
	// socket we get with net.ListenPacket wraps our data in IPv4 packets
	// already.  We do still need the IP layer to compute checksums
	// correctly, though.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}
	if err := gopacket.SerializeLayers(buf, opts, tcp); err != nil {
		log.Fatal(err)
	}

	conn, err := net.ListenPacket("ip4:tcp", "0.0.0.0")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("writing request")
	n, err := conn.WriteTo(buf.Bytes(), &net.IPAddr{IP: dstip})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Done - wrote %d bytes", n)
}
