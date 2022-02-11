package mcstatusgo

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	// packetID identifies the crafted packet as a status packet.
	packetID byte = 0
	// protocolVersion identifies the client's version of Minecraft (can be any valid protocol version).
	protocolVersion byte = 47
	// nextState is attached to the end of the handshake packet to signal a request for a status response from the server.
	nextState byte = 1
)

// requestPacket is the packet sent after the handshake to illicit a status response from the server.
var requestPacket []byte = []byte{nextState, packetID}

// Errors.
var (
	// ErrShortStatusResponse is returned when the received response is too small to contain valid data.
	ErrShortStatusResponse error = errors.New("invalid status response: response is too small")
	// ErrInvalidSizeInfo is returned when the information containing the JSON length does not match the actual JSON length.
	ErrInvalidSizeInfo error = errors.New("invalid status response: JSON size information is invalid")
	// ErrLargeVarInt is returned when a varint sent by the server is above the 5 bytes size limit.
	ErrLargeVarInt error = errors.New("invalid status response: varint sent by server exceeds size limit")
)

// ErrMissingInformation is used by both protocols and contains the specific value left out from the response.
type ErrMissingInformation struct {
	// Status or Query response.
	Protocol string
	// The value that was missing from the response.
	MissingValue string
}

func (e ErrMissingInformation) Error() string {
	return fmt.Sprintf("invalid %s response: %s missing from response.", e.Protocol, e.MissingValue)
}

// StatusResponse contains the information from the server status response.
type StatusResponse struct {
	// IP contains the server's IP.
	IP string

	// Port contains the server's port used for communication.
	Port uint16

	// Description contains a pretty-print JSON string of the server description.
	Description string `json:"-"`

	// Favicon contains the base64 encoded PNG image of the server that appears in the server list.
	Favicon string

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

		// Sample contains a random sample of players with their username and uuid currently on the server.
		Sample []map[string]string
	}

	ModInfo struct {
		// Type contains the server mod running on the server.
		Type string

		// ModList contains the plugins with their versions running on the server.
		ModList []map[string]string
	}
}

// Status requests basic server information from a Minecraft server.
//
// The Minecraft server must have SLP enabled.
//
// If a valid response is received, a StatusResponse is returned.
func Status(server string, port uint16, initialConnectionTimeout time.Duration, ioTimeout time.Duration) (StatusResponse, error) {
	serverAndPort := fmt.Sprintf("%s:%d", server, port)

	con, err := net.DialTimeout("tcp", serverAndPort, initialConnectionTimeout)
	if err != nil {
		return StatusResponse{}, err
	}
	// If the connection closes normally, this line will run but not do anything.
	defer resetConnection(con)

	serverIP := strings.Split(con.RemoteAddr().String(), ":")[0]

	err = initiateStatusRequest(con, ioTimeout, server, port)
	if err != nil {
		return StatusResponse{}, err
	}

	response, err := readStatusResponse(con, ioTimeout)
	if err != nil {
		return StatusResponse{}, err
	}

	con.Close()

	status, err := packageStatusResponse(serverIP, port, response)
	if err != nil {
		return StatusResponse{}, err
	}

	return status, nil
}

// resetConnection sends an RST packet to terminate the connection immediately.
func resetConnection(con net.Conn) {
	TCPCon := (con).(*net.TCPConn)
	TCPCon.SetLinger(0)
	TCPCon.Close()
}

// initiateStatusRequest handles sending the handshake and request packets.
func initiateStatusRequest(con net.Conn, timeout time.Duration, server string, port uint16) error {
	handshake := createStatusHandshakePacket(server, port)
	completedRequestPacket := append(handshake, requestPacket...)

	timeDeadline := time.Now().Add(timeout)
	con.SetWriteDeadline(timeDeadline)

	_, err := con.Write(completedRequestPacket)

	return err
}

// createStatusHandshakePacket crafts the handshake packet used to initialize the connection with the server.
func createStatusHandshakePacket(server string, port uint16) []byte {
	handshake := []byte{packetID, protocolVersion}
	handshake = append(handshake, serverToBytes(server)...)
	handshake = append(handshake, portToBytes(port)...)
	handshake = append(handshake, nextState)

	// Prepend handshake with varint containing the length of the handshake.
	handshake = append(writeVarInt(len(handshake)), handshake...)

	return handshake
}

// serverToBytes converts a server string into its []byte equivalent and prepends it with a varint containing its length.
func serverToBytes(server string) []byte {
	serverInBytes := []byte(server)
	serverLength := writeVarInt(len(serverInBytes))
	serverInBytesWithLength := append(serverLength, serverInBytes...)

	return serverInBytesWithLength
}

// portToBytes converts a uint16 port number to its []byte equivalent.
func portToBytes(port uint16) []byte {
	portInBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portInBytes, port)

	return portInBytes
}

// writeVarInt converts an int into its varint []byte equivalent.
func writeVarInt(number int) []byte {
	varInt := []byte{}

	for {
		var currentByte byte

		// No more bytes in the varint.
		if number&0xFFFFFF80 == 0 {
			currentByte = byte(number & 0x7F)
			varInt = append(varInt, currentByte)
			break
		}

		currentByte = byte((number & 0x7F) | 0x80)
		varInt = append(varInt, currentByte)

		number >>= 7
	}

	return varInt
}

// readStatusResponse receives the full status response from the server.
func readStatusResponse(con net.Conn, timeout time.Duration) ([]byte, error) {
	responseSize, err := readResponseSize(con, timeout)
	if err != nil {
		return nil, err
	}

	response := []byte{}

	timeDeadline := time.Now().Add(timeout)
	con.SetReadDeadline(timeDeadline)

	// Keep receiving bytes until the full message is received.
	for len(response) < responseSize {
		recvBuffer := make([]byte, 4096)
		bytesRead, err := con.Read(recvBuffer)

		if err != nil {
			return nil, err
		}

		response = append(response, recvBuffer[0:bytesRead]...)
	}

	return response, nil
}

// readResponseSize reads and parses the varint that prepends the server's response which contains the length of the response.
func readResponseSize(con net.Conn, timeout time.Duration) (int, error) {
	varInt := []byte{}
	timeDeadline := time.Now().Add(timeout)
	con.SetReadDeadline(timeDeadline)

	for {
		recvBuffer := make([]byte, 1)
		_, err := con.Read(recvBuffer)

		if err != nil {
			return -1, err
		}

		// Varint has terminated.
		if recvBuffer[0]&0x80 == 0 {
			varInt = append(varInt, recvBuffer[0])
			break
		}
		varInt = append(varInt, recvBuffer[0])
	}

	return readVarInt(varInt)
}

// readVarInt converts a varint into its int equivalent.
func readVarInt(varInt []byte) (int, error) {
	number := 0
	bitOffSet := 0

	for _, currentByte := range varInt {
		if bitOffSet == 35 {
			return -1, ErrLargeVarInt
		}

		number |= int(currentByte&0x7F) << bitOffSet

		if currentByte&0x80 == 0 {
			break
		}
		bitOffSet += 7
	}

	return number, nil
}

// packageStatusResponse formats, parses, and packages the response into status.
func packageStatusResponse(serverIP string, port uint16, response []byte) (StatusResponse, error) {
	status := StatusResponse{}
	status.IP = serverIP
	status.Port = port

	formatedResponse, err := formatStatusResponse(response)
	if err != nil {
		return StatusResponse{}, err
	}

	// Return an error if the received response is missing information.
	err = validateStatusResponse(formatedResponse)
	if err != nil {
		return StatusResponse{}, err
	}

	// Unmarshal the formatted JSON response into status.
	err = json.Unmarshal(formatedResponse, &status)
	if err != nil {
		return StatusResponse{}, err
	}

	// Add the description information to status.
	err = packageDescription(formatedResponse, &status)
	if err != nil {
		return StatusResponse{}, err
	}

	return status, nil
}

// formatResponse cleans the response for JSON processing.
func formatStatusResponse(response []byte) ([]byte, error) {
	if len(response) < 4 {
		return nil, ErrShortStatusResponse
	}

	// Remove stateID byte
	response = response[1:]

	// Get varint that contains length of JSON string.
	jsonLen := []byte{}
	for _, currentByte := range response {
		if currentByte&0x80 == 0 {
			jsonLen = append(jsonLen, currentByte)
			break
		}
		jsonLen = append(jsonLen, currentByte)
	}

	// Remove varint that precedes the JSON string.
	response = response[len(jsonLen):]

	// Parse JSON string length to an int.
	jsonLength, err := readVarInt(jsonLen)
	if err != nil {
		return nil, err
	}

	// Check if JSON size information matches the size of the JSON string.
	if jsonLength != len(response) {
		return nil, ErrInvalidSizeInfo
	}

	return response, nil
}

// validateStatusResponse checks for missing information from the status response.
func validateStatusResponse(response []byte) error {
	// The players sample, favicon, and modinfo fields are not included in the validation because they are all optional.
	var verifyResponse struct {
		Description interface{}
		Players     struct{ Max, Online interface{} }
		Version     struct{ Name, Protocol interface{} }
	}

	err := json.Unmarshal(response, &verifyResponse)
	if err != nil {
		return err
	}

	// Check if any of the values were left out from the status response.
	if verifyResponse.Description == nil {
		return ErrMissingInformation{"status", "description"}
	}
	if verifyResponse.Players.Max == nil {
		return ErrMissingInformation{"status", "max players"}
	}
	if verifyResponse.Players.Online == nil {
		return ErrMissingInformation{"status", "online players"}
	}
	if verifyResponse.Version.Name == nil {
		return ErrMissingInformation{"status", "version name"}
	}
	if verifyResponse.Version.Protocol == nil {
		return ErrMissingInformation{"status", "version protocol"}
	}

	return nil
}

// packageDescription parses the description into a pretty-print JSON string and packages it into status.
func packageDescription(response []byte, status *StatusResponse) error {
	var descriptionInfo struct {
		Description interface{}
	}

	err := json.Unmarshal(response, &descriptionInfo)
	if err != nil {
		return err
	}

	descJSONBytes, err := json.MarshalIndent(descriptionInfo.Description, "", "  ")
	if err != nil {
		return err
	}

	status.Description = string(descJSONBytes)

	return nil
}