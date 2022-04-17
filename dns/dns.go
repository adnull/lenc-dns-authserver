package dnsserv

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
)

// DNSHeader describes the request/response DNS header
type DNSHeader struct {
	TransactionID  uint16
	Flags          uint16
	NumQuestions   uint16
	NumAnswers     uint16
	NumAuthorities uint16
	NumAdditionals uint16
}

// DNSResourceRecord describes individual records in the request and response of the DNS payload body
type DNSResourceRecord struct {
	DomainName         string
	Type               uint16
	Class              uint16
	TimeToLive         uint32
	ResourceDataLength uint16
	ResourceData       []byte
}

const (
	TypeA                  uint16 = 1  // a host address
	TypeTXT                uint16 = 16 // a host address
	TypeSOA                uint16 = 6  // a host address
	ClassINET              uint16 = 1  // the Internet
	FlagResponse           uint16 = 1 << 15
	UDPMaxMessageSizeBytes uint   = 512 // RFC1035
)

func readDomainName(requestBuffer *bytes.Buffer) (string, error) {
	var domainName string

	b, err := requestBuffer.ReadByte()

	for ; b != 0 && err == nil; b, err = requestBuffer.ReadByte() {
		labelLength := int(b)
		labelBytes := requestBuffer.Next(labelLength)
		labelName := string(labelBytes)

		if len(domainName) == 0 {
			domainName = labelName
		} else {
			domainName += "." + labelName
		}
	}

	return domainName, err
}

func handleDNSClient(n int, request []byte, sc *net.UDPConn, ca *net.UDPAddr) {

	var requestBuffer = bytes.NewBuffer(request)
	var queryHeader DNSHeader
	var queryResourceRecords []DNSResourceRecord

	err := binary.Read(requestBuffer, binary.BigEndian, &queryHeader) // network byte order is big endian

	if err != nil {
		fmt.Println("Error decoding header: ", err.Error())
	}

	queryResourceRecords = make([]DNSResourceRecord, queryHeader.NumQuestions)

	for idx, _ := range queryResourceRecords {
		queryResourceRecords[idx].DomainName, err = readDomainName(requestBuffer)

		if err != nil {
			fmt.Println("Error decoding label: ", err.Error())
		}

		queryResourceRecords[idx].Type = binary.BigEndian.Uint16(requestBuffer.Next(2))
		queryResourceRecords[idx].Class = binary.BigEndian.Uint16(requestBuffer.Next(2))
		log.Print(queryResourceRecords[idx])
	}

}

func ServeDNS(serverAddress string) {
	sa, err := net.ResolveUDPAddr("udp", serverAddress)
	if err != nil {
		log.Printf("Error resolving address %s", serverAddress)
		os.Exit(1)
	}

	serverConn, err := net.ListenUDP("udp", sa)
	if err != nil {
		log.Printf("Error listening: %s", err.Error())
		os.Exit(1)
	}

	log.Printf("DNS server listening on %s", serverAddress)
	defer serverConn.Close()

	for {
		requestBytes := make([]byte, UDPMaxMessageSizeBytes)

		n, clientAddr, err := serverConn.ReadFromUDP(requestBytes)

		if err != nil {
			log.Println("Error receiving: ", err.Error())
		} else {
			log.Println("Received request from ", clientAddr)
			go handleDNSClient(n, requestBytes, serverConn, clientAddr) // array is value type (call-by-value), i.e. copied
		}
	}
}
