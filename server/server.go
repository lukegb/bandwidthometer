package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DstPort       = 31814
	Interface     = "vlan2"
	StatsInterval = 1 * time.Second
)

var (
	DstAddr = net.IPv4(172, 27, 27, 255)

	ErrInterfaceMismatch = errors.New(`interfaces do not match`)
)

type InterfaceStats struct {
	Interface    string
	TimeRecorded time.Time

	RxBytes uint64
	TxBytes uint64
}

func (is1 *InterfaceStats) Sub(is2 *InterfaceStats) (*InterfaceDelta, error) {
	if is1.Interface != is2.Interface {
		return nil, ErrInterfaceMismatch
	}

	if is1.TimeRecorded.After(is2.TimeRecorded) {
		is1, is2 = is2, is1
	}

	timeDelta := is2.TimeRecorded.Sub(is1.TimeRecorded)
	tdSeconds := timeDelta.Seconds()

	id := new(InterfaceDelta)
	id.Interface = is1.Interface
	id.RxBytesPerSecond = int32(is2.RxBytes-is1.RxBytes) / int32(tdSeconds)
	if is2.RxBytes < is1.RxBytes {
		id.RxBytesPerSecond = -1 * id.RxBytesPerSecond
	}
	id.TxBytesPerSecond = int32(is2.TxBytes-is1.TxBytes) / int32(tdSeconds)
	if is2.TxBytes < is1.TxBytes {
		id.TxBytesPerSecond = -1 * id.TxBytesPerSecond
	}

	return id, nil
}

type InterfaceDelta struct {
	Interface string

	RxBytesPerSecond int32
	TxBytesPerSecond int32
}

func (id InterfaceDelta) String() string {
	return fmt.Sprintf("[%s] RX: %d bps, TX: %d bps", id.Interface, id.RxBytesPerSecond, id.TxBytesPerSecond)
}

func readUintFromFile(filename string) (uint64, error) {
	bytesval, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	strval := strings.TrimSpace(string(bytesval))

	return strconv.ParseUint(strval, 10, 64)
}

func gatherInterfaceData(intfName string) (*InterfaceStats, error) {
	is := new(InterfaceStats)

	intfDir := filepath.Join("/sys/class/net", intfName)
	intfStatsDir := filepath.Join(intfDir, "statistics")

	var err error
	is.RxBytes, err = readUintFromFile(filepath.Join(intfStatsDir, "rx_bytes"))
	if err != nil {
		return nil, err
	}

	is.TxBytes, err = readUintFromFile(filepath.Join(intfStatsDir, "tx_bytes"))
	if err != nil {
		return nil, err
	}

	is.TimeRecorded = time.Now()
	is.Interface = intfName

	return is, nil
}

func packAndSend(intfDelta *InterfaceDelta, w io.Writer) error {
	// I want to send this in one packet, so we have to construct this into a binary array and THEN send it
	buf := new(bytes.Buffer)

	// RX
	if err := binary.Write(buf, binary.LittleEndian, intfDelta.RxBytesPerSecond); err != nil {
		return err
	}

	// TX
	if err := binary.Write(buf, binary.LittleEndian, intfDelta.TxBytesPerSecond); err != nil {
		return err
	}

	_, err := buf.WriteTo(w)
	return err
}

func main() {
	// check to see if we can find /sys/class/net/<intf>/statistics/{rx_bytes/tx_bytes}
	intfData, err := gatherInterfaceData(Interface)
	if err != nil {
		log.Fatal("Error gathering interface data for ", Interface, ": ", err)
	}

	log.Println("Initial readings for:", intfData.Interface)
	log.Println("            TX Bytes:", intfData.TxBytes)
	log.Println("            RX Bytes:", intfData.RxBytes)
	log.Println()

	log.Println("Sending to", DstAddr, "port", DstPort)
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   DstAddr,
		Port: DstPort,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	for {
		// using a sleep here instead of a ticker
		// to avoid zero-second intervals if this loop
		// takes no time to execute
		time.Sleep(StatsInterval)

		newIntfData, err := gatherInterfaceData(Interface)
		if err != nil {
			log.Println("Got error getting interface data, skipping this round:", err)
			continue
		}

		intfDelta, err := newIntfData.Sub(intfData)
		if err != nil {
			log.Fatal("Got error computing delta, bailing! ", err)
		}

		if intfDelta.RxBytesPerSecond < 0 || intfDelta.TxBytesPerSecond < 0 {
			log.Println("RX/TX bytes went backwards, ignoring this round")
			intfData = newIntfData
			continue
		}

		log.Println(intfDelta)
		if err := packAndSend(intfDelta, conn); err != nil {
			log.Println("Error sending delta to UDP:", err)
		}
		intfData = newIntfData
	}
}
