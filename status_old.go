package mcstatusgo

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// This file contains all older implementations of the status protocol.

/* Status Legacy */

var (
	// legacyRequestPacket is the packet sent to elicit a legacy status response from the server.
	legacyRequestPacket []byte = []byte{0xFE, 0x01, 0xFA}
)

// Errors.
var (
	// ErrShortStatusLegacyResponse is returned when the received response is too small to contain valid data.
	ErrShortStatusLegacyResponse error = errors.New("invalid status legacy response: response is too small to contain valid data")
	// ErrStatusLegacyMissingInformation is returned when the received response doesn't contain all 5 expected values.
	ErrStatusLegacyMissingInformation error = errors.New("invalid status legacy response: response doesn't contain all 5 expected values")
)

// StatusLegacyResponse contains the information from the legacy status request.
// https://wiki.vg/Server_List_Ping#Server_to_client
type StatusLegacyResponse struct {
	// IP contains the server's IP.
	IP string

	// Port contains the server's port used for communication.
	Port uint16

	// Latency contains the duration of time waited for the response.
	Latency time.Duration

	// Description contains the MOTD of the server.
	Description string

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

// StatusLegacy requests basic server information from a Minecraft server using the older legacy implementation of Status.
//
// The Minecraft server must have SLP enabled.
//
// If a valid response is received, a StatusLegacyResponse is returned.
// https://wiki.vg/Server_List_Ping#1.6
func StatusLegacy(server string, port uint16, initialConnectionTimeout time.Duration, ioTimeout time.Duration) (StatusLegacyResponse, error) {
	serverAndPort := fmt.Sprintf("%s:%d", server, port)

	con, err := net.DialTimeout("tcp", serverAndPort, initialConnectionTimeout)
	if err != nil {
		return StatusLegacyResponse{}, err
	}
	// If the connection closes normally, this line will run but not do anything.
	defer resetConnection(con)

	// Split the string "IP:PORT" by : to get the IP of the remote host.
	serverIP := strings.Split(con.RemoteAddr().String(), ":")[0]

	err = initiateRequest(con, ioTimeout, legacyRequestPacket)
	if err != nil {
		return StatusLegacyResponse{}, err
	}

	response, latency, err := readLegacyStatusResponse(con, ioTimeout)
	if err != nil {
		return StatusLegacyResponse{}, err
	}

	con.Close()

	statusLegacy, err := packageLegacyStatusResponse(serverIP, port, latency, response)
	if err != nil {
		return StatusLegacyResponse{}, err
	}

	return statusLegacy, nil
}

// readLegacyStatusResponse receives the full legacy status response from the server.
func readLegacyStatusResponse(con net.Conn, timeout time.Duration) ([]byte, time.Duration, error) {
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

// packageLegacyStatusResponse parses and packages the response into statusLegacy.
func packageLegacyStatusResponse(serverIP string, port uint16, latency time.Duration, response []byte) (StatusLegacyResponse, error) {
	statusLegacy := StatusLegacyResponse{}
	statusLegacy.IP = serverIP
	statusLegacy.Port = port
	statusLegacy.Latency = latency

	responseList, err := parseLegacyStatusResponse(response)
	if err != nil {
		return StatusLegacyResponse{}, err
	}

	err = packageLegacyStatusValues(responseList, &statusLegacy)
	if err != nil {
		return StatusLegacyResponse{}, err
	}

	return statusLegacy, nil
}

// parseLegacyStatusResponse parses the doubly null-terminated byte string values into a []string.
func parseLegacyStatusResponse(response []byte) ([]string, error) {
	if len(response) < 10 {
		return nil, ErrShortQueryResponse
	}

	// Remove the bytes that prepend the response.
	response = response[9:]

	responseList := []string{}
	currentValue := []byte{}

	// appendValue is set to true when the read byte is 0. If the subsequent byte is 0, the value is appended and appendValue is set to false.
	var appendValue bool
	for _, currentByte := range response {
		if currentByte == 0 {
			// Last byte was 0, so append this value.
			if appendValue {
				responseList = append(responseList, string(currentValue))
				currentValue = []byte{}
				appendValue = false
			} else {
				appendValue = true
			}
		} else {
			currentValue = append(currentValue, currentByte)
			appendValue = false
		}
	}

	// Append the final value that wasn't terminated with two null characters.
	responseList = append(responseList, string(currentValue))

	return responseList, nil
}

// packageLegacyStatusValues takes responseList and parses and packages the values into statusLegacy.
func packageLegacyStatusValues(responseList []string, statusLegacy *StatusLegacyResponse) error {
	if len(responseList) < 5 {
		return ErrStatusLegacyMissingInformation
	}

	// Package the string values.
	statusLegacy.Version.Name = responseList[1]
	statusLegacy.Description = responseList[2]

	// Convert and package the int values.
	protocolVersion, err := stringToInt(responseList[0])
	if err != nil {
		return err
	}
	statusLegacy.Version.Protocol = protocolVersion

	playersOnline, err := stringToInt(responseList[3])
	if err != nil {
		return err
	}
	statusLegacy.Players.Online = playersOnline

	playersMax, err := stringToInt(responseList[4])
	if err != nil {
		return err
	}
	statusLegacy.Players.Max = playersMax

	return nil
}

/* Status Beta */

const (
	// betaRequestPacket is the packet sent to elicit a beta status response from the server.
	betaRequestPacket byte = 0xFE
)

// StatusBetaResponse contains the information from the beta status request.
type StatusBetaResponse struct {
	// IP contains the server's IP.
	IP string

	// Port contains the server's port used for communication.
	Port uint16

	// Latency contains the duration of time waited for the response.
	Latency time.Duration

	// Description contains the MOTD of the server.
	Description string

	Players struct {
		// Max contains the maximum number of players the server supports.
		Max int

		// Online contains the current number of players on the server.
		Online int
	}
}

// StatusBeta requests basic server information from a Minecraft server using the beta (oldest version) implementation of Status.
//
// The Minecraft server must have SLP enabled.
//
// If a valid response is received, a StatusBetaResponse is returned.
// https://wiki.vg/Server_List_Ping#Beta_1.8_to_1.3
func StatusBeta(server string, port uint16, initialConnectionTimeout time.Duration, ioTimeout time.Duration) (StatusBetaResponse, error) {
	serverAndPort := fmt.Sprintf("%s:%d", server, port)

	con, err := net.DialTimeout("tcp", serverAndPort, initialConnectionTimeout)
	if err != nil {
		return StatusBetaResponse{}, err
	}
	// If the connection closes normally, this line will run but not do anything.
	defer resetConnection(con)

	// Split the string "IP:PORT" by : to get the IP of the remote host.
	// serverIP := strings.Split(con.RemoteAddr().String(), ":")[0]

	err = initiateRequest(con, ioTimeout, []byte{betaRequestPacket})
	if err != nil {
		return StatusBetaResponse{}, err
	}

	_, err = readBetaStatusResponse(con, ioTimeout)
	if err != nil {
		return StatusBetaResponse{}, err
	}

	con.Close()

	// Process received response here

	return StatusBetaResponse{}, nil
}

// readStatusResponse receives the full status response from the server.
func readBetaStatusResponse(con net.Conn, timeout time.Duration) ([]byte, error) {
	responseSize, err := readBetaStatusResponseSize(con, timeout)
	if err != nil {
		return nil, err
	}

	response := []byte{}

	// Keep receiving bytes until the full message is received.
	setDeadline(&con, timeout)
	for len(response) < responseSize {
		recvBuffer := make([]byte, 32)
		bytesRead, err := con.Read(recvBuffer)

		if err != nil {
			return nil, err
		}

		response = append(response, recvBuffer[0:bytesRead]...)
	}

	return response, nil
}

// readBetaStatusResponseSize reads and parses the short that prepends the server's response which contains the length of the response.
func readBetaStatusResponseSize(con net.Conn, timeout time.Duration) (int, error) {
	response := make([]byte, 3)

	_, err := con.Read(response)
	if err != nil {
		return -1, err
	}

	// Remove the kick packet from the front.
	response = response[1:]

	// For unknown reasons (most likely due to encoding), the response size must be multiplied by 2 to contain the actual response length.
	responseSize := int(binary.BigEndian.Uint16(response)) * 2

	return responseSize, nil
}