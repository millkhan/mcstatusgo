package mcstatusgo

import (
	"fmt"
	"net"
	"time"
)

// This file contains all older implementations of the status protocol.

var (
	legacyRequestPacket []byte = []byte{0xFE, 0x01, 0xFA}
)

type StatusLegacyResponse struct {
	// IP contains the server's IP.
	IP string

	// Port contains the server's port used for communication.
	Port uint16

	// Latency contains the duration of time waited for the response.
	Latency time.Duration

	// Description contains a pretty-print JSON string of the server description.
	Description string `json:"-"`

	Version struct {
		// Name contains the version of Minecraft running on the server.
		Name string

		// Protocol contains the protocol version used in the request or that should be used when connecting to the server.
		Protocol int
	}

	Players struct {
		// Max contains the maximum number of players the server supports.
		Max int

		// Online contains the current number of players on the server.
		Online int
	}
}

func StatusLegacy(server string, port uint16, initialConnectionTimeout time.Duration, ioTimeout time.Duration) error {
	serverAndPort := fmt.Sprintf("%s:%d", server, port)

	con, err := net.DialTimeout("tcp", serverAndPort, initialConnectionTimeout)
	if err != nil {
        return err
	}
	// If the connection closes normally, this line will run but not do anything.
	defer resetConnection(con)

	// serverIP := strings.Split(con.RemoteAddr().String(), ":")[0]

	err = initiateStatusLegacyRequest(con, ioTimeout, server, port)
	if err != nil {
        return err
	}

	_, _, err = readStatusLegacyResponse(con, ioTimeout)
	if err != nil {
        return err
	}

	con.Close()

    return nil
    // Parsing and packaging done below...
}

func initiateStatusLegacyRequest(con net.Conn, timeout time.Duration, server string, port uint16) error {
	setDeadline(&con, timeout)
	_, err := con.Write(legacyRequestPacket)

	return err
}

func readStatusLegacyResponse(con net.Conn, timeout time.Duration) ([]byte, time.Duration, error) {
	response := make([]byte, 512)
	setDeadline(&con, timeout)

	startTime := time.Now()
	bytesRead, err := con.Read(response)
	if err != nil {
		return nil, -1, err
	}
	latency := time.Since(startTime)

	response = response[0:bytesRead]

	return response, latency, nil
}