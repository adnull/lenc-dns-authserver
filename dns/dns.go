package dnsserv

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
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

func writeDomainName(domain *DNSResourceRecord, b bytes.Buffer) error {

	parts := strings.Split(domain.DomainName, ".")
	for _, p := range parts {
		b.WriteByte(uint8(len(p)))
		b.WriteString(p)
	}
	b.WriteByte(0)
	//	b.Write()
	return nil
}

func makeAnswers(r []DNSResourceRecord) ([]DNSResourceRecord, error) {

	var answers []DNSResourceRecord

	for i, req := range r {
		fmt.Println("l", i, req)
		if req.Type == TypeTXT {
			var a = "ZXC22"
			answers = append(answers, DNSResourceRecord{req.DomainName, TypeTXT, req.Class, 1, uint16(len(a)), []byte(a)})
		}
	}

	fmt.Println("answer:", answers)
	return answers, nil

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

	answers, _ := makeAnswers(queryResourceRecords)

	var aflags uint16 = 0

	error := 0
	if len(answers) > 0 {
		error = 3
	}
	bits := "100001010000" + fmt.Sprintf("%04b", error)
	fmt.Println("bits", bits)
	flags, _ := strconv.ParseUint(bits, 2, 16)
	fmt.Println("flags=", flags)
	aflags = uint16(flags)
	//	aheader := DNSHeader{queryHeader.TransactionID, aflags, queryHeader.NumQuestions, uint16(len(answers)), 0, 0}
	aheader := DNSHeader{queryHeader.TransactionID, aflags, queryHeader.NumQuestions, 0, 0, 0}

	var binbuff bytes.Buffer
	enc := gob.NewEncoder(&binbuff)

	if err = enc.Encode(aheader); err == nil {

		var ansbuff bytes.Buffer
		//		if err = binary.Write(&ansbuff, binary.BigEndian, binbuff); err == nil {

		if _, err = sc.WriteToUDP(binbuff.Bytes(), ca); err == nil {
			fmt.Println("header=", aheader)
			for _, a := range answers {
				binbuff.Reset()
				ansbuff.Reset()
				if err = enc.Encode(a); err == nil {
					//					if err = binary.Write(&ansbuff, binary.BigEndian, binbuff); err == nil {
					if _, err = sc.WriteToUDP(binbuff.Bytes(), ca); err == nil {
						fmt.Println("answer=", a)
					} else {
						fmt.Println(err)
					}
				}
			}
		} else {
			fmt.Println(err)
		}

	} else {
		fmt.Println(err)
	}

	//	if err = binary.Write(&binbuff, binary.BigEndian, aheader); err == nil {
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
