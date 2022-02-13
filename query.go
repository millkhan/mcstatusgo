package mcstatusgo

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	// handshakeByte identifies the crafted packet as a handshake packet.
	handshakeByte byte = 9
	// statByte identifies the packet as a request for query information.
	statByte byte = 0
)

var (
	// magicBytes must prepend every message sent to the server.
	magicBytes []byte = []byte{0xFE, 0xFD}
	// fullQueryPadding is added at the end of the request packet to indicate a request for full query information.
	fullQueryPadding []byte = []byte{0, 0, 0, 0}
	// playerSplit is the token used to split the full query response into two parts for parsing.
	playerToken []byte = []byte{0, 1, 112, 108, 97, 121, 101, 114, 95, 0, 0}
)

// Errors.
var (
	// ErrShortQueryResponse is returned when the received response is too small to contain valid data.
	ErrShortQueryResponse error = errors.New("invalid query response: response is too small")
	// ErrShortChallengeToken is returned when the received challenge token is too small to be valid.
	ErrShortChallengeToken error = errors.New("invalid query response: challenge token is too small")
	// ErrAbsentChallengeTokenNullTerminator is returned when the challenge token doesn't contain a null-terminator at the end.
	ErrAbsentChallengeTokenNullTerminator = errors.New("invalid query response: challenge token doesn't contain a null-terminator")
	// ErrAbsentPlayerToken is returned when the player token used to split the full query response into two parts for parsing isn't present.
	ErrAbsentPlayerToken error = errors.New("invalid query response: player token not in response")
)

// FullQueryResponse contains the information from the full query request.
type FullQueryResponse struct {
	// IP contains the server's IP.
	IP string

	// Port contains the server's port used for communication.
	Port uint16

	// Description contains the MOTD of the server.
	Description string

	// Gametype contains a string which is usually 'SMP'.
	GameType string

	// GameID contains a string which is usually 'MINECRAFT'.
	GameID string

	// MapName contains the name of the map running on the server.
	MapName string

	Version struct {
		// Name contains the version of Minecraft running on the server.
		Name string
	}

	Players struct {
		// Max contains the maximum number of players the server supports.
		Max int

		// Online contains the current number of players on the server.
		Online int

		// PlayerList contains the usernames of the players currently on the server.
		PlayerList []string
	}

	ModInfo struct {
		// Type contains the server mod running on the server.
		Type string

		// ModList contains the plugins with their versions running on the server.
		ModList []map[string]string
	}
}

// FullQuery requests detailed server information from a Minecraft server.
//
// The Minecraft server must have the "enable-query" property set to true.
//
// If a valid response is received, a FullQueryResponse is returned.
func FullQuery(server string, port uint16, initialConnectionTimeout time.Duration, ioTimeout time.Duration) (FullQueryResponse, error) {
	serverAndPort := fmt.Sprintf("%s:%d", server, port)

	con, err := net.DialTimeout("udp", serverAndPort, initialConnectionTimeout)
	if err != nil {
		return FullQueryResponse{}, err
	}
	// If the connection closes normally, this line will run but not do anything.
	defer con.Close()

	serverIP := strings.Split(con.RemoteAddr().String(), ":")[0]

	err = initiateQueryRequest(con, ioTimeout)
	if err != nil {
		return FullQueryResponse{}, err
	}

	response, err := readQueryResponse(con, ioTimeout)
	if err != nil {
		return FullQueryResponse{}, err
	}

	con.Close()

	fullQuery, err := packageFullQueryResponse(serverIP, port, response)
	if err != nil {
		return FullQueryResponse{}, err
	}

	return fullQuery, nil
}

// initiateQueryRequest handles sending the handshake and request packets.
func initiateQueryRequest(con net.Conn, timeout time.Duration) error {
	sessionID := createSessionID()
	handshake := createQueryHandshakePacket(sessionID)

	challengeToken, err := readChallengeToken(con, timeout, handshake)
	if err != nil {
		return err
	}

	err = sendFullQueryRequest(con, timeout, sessionID, challengeToken)
	if err != nil {
		return err
	}

	return nil
}

// createSessionID creates a random sessionID for the query request.
func createSessionID() []byte {
	rand.Seed(time.Now().UnixNano())
	sessionID := make([]byte, 4)

	randomSessionID := 0x0F0F0F0F & rand.Int()
	binary.BigEndian.PutUint32(sessionID, uint32(randomSessionID))

	return sessionID
}

// createQueryHandshakePacket crafts the handshake packet used to initiate the request.
func createQueryHandshakePacket(sessionID []byte) []byte {
	handshake := []byte(magicBytes)
	handshake = append(handshake, handshakeByte)
	handshake = append(handshake, sessionID...)

	return handshake
}

// readChallengeToken reads and parses the challenge token sent by the server.
func readChallengeToken(con net.Conn, timeout time.Duration, handshake []byte) ([]byte, error) {
	timeDeadline := time.Now().Add(timeout)
	con.SetWriteDeadline(timeDeadline)

	_, err := con.Write(handshake)
	if err != nil {
		return nil, err
	}

	timeDeadline = time.Now().Add(timeout)
	con.SetReadDeadline(timeDeadline)

	potentialChallengeToken := make([]byte, 32)
	numRead, err := con.Read(potentialChallengeToken)
	if err != nil {
		return nil, err
	}
	potentialChallengeToken = potentialChallengeToken[0:numRead]

	challengeToken, err := parseChallengeToken(potentialChallengeToken)
	if err != nil {
		return nil, err
	}

	return challengeToken, nil
}

// parseChallengeToken parses the cleaned challenge token into an int represented in []byte.
func parseChallengeToken(potentialChallengeToken []byte) ([]byte, error) {
	potentialChallengeToken, err := cleanChallengeToken(potentialChallengeToken)
	if err != nil {
		return []byte{}, err
	}

	var challengeTokenInt int64
	challengeTokenBytes := make([]byte, 4)
	challengeTokenString := string(potentialChallengeToken)

	// If challenge token is negative.
	if challengeTokenString[0] == '-' {
		challengeTokenInt, err = strconv.ParseInt(challengeTokenString[1:], 10, 32)
		if err != nil {
			return []byte{}, err
		}
		challengeTokenInt *= -1
	} else {
		challengeTokenInt, err = strconv.ParseInt(challengeTokenString, 10, 32)
		if err != nil {
			return []byte{}, err
		}
	}
	binary.BigEndian.PutUint32(challengeTokenBytes, uint32(challengeTokenInt))

	return challengeTokenBytes, nil
}

// cleanChallengeToken formats and checks the received challenge token.
func cleanChallengeToken(potentialChallengeToken []byte) ([]byte, error) {
	if len(potentialChallengeToken) < 7 {
		return nil, ErrShortChallengeToken
	}

	// Remove Type and sessionID bytes from the beginning.
	potentialChallengeToken = potentialChallengeToken[5:]

	// Return an error if the challenge token doesn't have a null-terminator at the end.
	if potentialChallengeToken[len(potentialChallengeToken)-1] != 0 {
		return nil, ErrAbsentChallengeTokenNullTerminator
	}

	// Remove any lingering null-terminators.
	cleanedToken := []byte{}
	for _, currentByte := range potentialChallengeToken {
		if currentByte != 0 {
			cleanedToken = append(cleanedToken, currentByte)
		}
	}

	return cleanedToken, nil
}

// sendFullQueryRequest sends the full query request packet to the server.
func sendFullQueryRequest(con net.Conn, timeout time.Duration, sessionID []byte, challengeToken []byte) error {
	fullQueryRequestPacket := createFullQueryRequestPacket(sessionID, challengeToken)

	timeDeadline := time.Now().Add(timeout)
	con.SetWriteDeadline(timeDeadline)

	_, err := con.Write(fullQueryRequestPacket)
	return err
}

// createFullQueryRequestPacket uses the information received from the handshake to create the full query request packet.
func createFullQueryRequestPacket(sessionID []byte, challengeToken []byte) []byte {
	fullQueryRequestPacket := append(magicBytes, statByte)
	fullQueryRequestPacket = append(fullQueryRequestPacket, sessionID...)
	fullQueryRequestPacket = append(fullQueryRequestPacket, challengeToken...)
	fullQueryRequestPacket = append(fullQueryRequestPacket, fullQueryPadding...)

	return fullQueryRequestPacket
}

// readQueryResponse receives the query response.
func readQueryResponse(con net.Conn, timeout time.Duration) ([]byte, error) {
	timeDeadline := time.Now().Add(timeout)
	con.SetReadDeadline(timeDeadline)

	response := make([]byte, 8192)
	bytesRead, err := con.Read(response)
	if err != nil {
		return nil, err
	}
	response = response[0:bytesRead]

	return response, nil
}

// packageFullQueryResponse parses and packages the response into fullQuery.
func packageFullQueryResponse(serverIP string, port uint16, response []byte) (FullQueryResponse, error) {
	fullQuery := FullQueryResponse{}
	fullQuery.IP = serverIP
	fullQuery.Port = port

	// Split the response using the player token into a key value section and a null-terminated string section containing the players online for parsing.
	splitResponse := bytes.Split(response, playerToken)
	if len(splitResponse) != 2 {
		return FullQueryResponse{}, ErrAbsentPlayerToken
	}

	keyValueSection := splitResponse[0]
	playerSection := splitResponse[1]

	responseMapBytes, err := parseKeyValueSection(keyValueSection)
	if err != nil {
		return FullQueryResponse{}, err
	}

	err = validateQueryResponse(responseMapBytes)
	if err != nil {
		return FullQueryResponse{}, err
	}

	err = packageKeyValueSection(responseMapBytes, &fullQuery)
	if err != nil {
		return FullQueryResponse{}, err
	}

	packagePlayerSection(playerSection, &fullQuery)

	return fullQuery, nil
}

// parseKeyValueSection parses the key mapped values from the full query response into a JSON []byte.
func parseKeyValueSection(keyValueSection []byte) ([]byte, error) {
	if len(keyValueSection) < 16 {
		return nil, ErrShortQueryResponse
	}

	// Remove type, sessionID, and padding bytes from the front.
	keyValueSection = keyValueSection[16:]

	// Parse each key and its corresponding value and insert it into responseMap.
	var currentValue []byte
	var keyValue string
	responseMap := make(map[string]string)
	isKey := true
	for _, currentByte := range keyValueSection {
		// The current byte string being read has terminated.
		if currentByte == 0 {
			// Keep the key value until its value has been parsed.
			if isKey {
				keyValue = string(currentValue)
				currentValue = []byte{}
				isKey = false
			} else {
				// Map the stored key to the read value.
				responseMap[keyValue] = string(currentValue)
				currentValue = []byte{}
				isKey = true
			}
		} else {
			currentValue = append(currentValue, currentByte)
		}
	}

	responseMapBytes, err := json.Marshal(responseMap)
	if err != nil {
		return nil, err
	}

	return responseMapBytes, nil
}

// validateQueryResponse checks for missing information from the query response.
func validateQueryResponse(responseMapBytes []byte) error {
	var verifyResponse struct {
		Hostname, Gametype, Game_id, Version, Plugins, Map, Numplayers, Maxplayers interface{}
	}

	err := json.Unmarshal(responseMapBytes, &verifyResponse)
	if err != nil {
		return err
	}

	values := reflect.ValueOf(verifyResponse)
	for i := 0; i < values.NumField(); i++ {
		valueType := values.Field(i).Interface()
		valueName := strings.ToLower(values.Type().Field(i).Name)

		// A value was left out from query response.
		if valueType == nil {
			return ErrMissingInformation{"query", valueName}
		}
	}

	return nil
}

// packageKeyValueSection manually unmarshals and packages the key value section into fullQuery to preserve an identitical structure to StatusResponse{}.
func packageKeyValueSection(responseMapBytes []byte, fullQuery *FullQueryResponse) error {
	var keyValueInfo struct {
		Maxplayers, Numplayers                             int `json:",string"`
		Hostname, Gametype, Game_id, Map, Version, Plugins string
	}

	err := json.Unmarshal(responseMapBytes, &keyValueInfo)
	if err != nil {
		return err
	}

	fullQuery.Players.Max = keyValueInfo.Maxplayers
	fullQuery.Players.Online = keyValueInfo.Numplayers
	fullQuery.Description = keyValueInfo.Hostname
	fullQuery.GameType = keyValueInfo.Gametype
	fullQuery.GameID = keyValueInfo.Game_id
	fullQuery.MapName = keyValueInfo.Map
	fullQuery.Version.Name = keyValueInfo.Version
	packagePluginSection(keyValueInfo.Plugins, fullQuery)

	return nil
}

// packagePluginSection parses and packages the plugin section into fullQuery.
func packagePluginSection(pluginSection string, fullQuery *FullQueryResponse) {
	// The server is vanilla or doesn't send plugin information.
	if len(pluginSection) == 0 {
		return
	}

	pluginSectionSplit := strings.SplitN(pluginSection, ": ", 2)
	serverModName := pluginSectionSplit[0]

	// Only the server mod name was in the response.
	if len(pluginSectionSplit) == 1 {
		fullQuery.ModInfo.Type = serverModName
		return
	}

	pluginList := []map[string]string{}
	pluginString := pluginSectionSplit[1]

	if pluginString != "" {
		// Split the plugin list into a slice of strings which each contain their name and version split by a whitespace.
		pluginStringSplit := strings.Split(pluginString, "; ")

		for _, plugin := range pluginStringSplit {
			var pluginName string
			var pluginVersion string

			pluginSplit := strings.Split(plugin, " ")

			// Plugin with no version provided.
			if len(pluginSplit) == 1 {
				pluginName = pluginSplit[0]
			} else {
				pluginName = pluginSplit[0]
				pluginVersion = pluginSplit[1]
			}

			completedPlugin := make(map[string]string)
			completedPlugin[pluginName] = pluginVersion
			pluginList = append(pluginList, completedPlugin)
		}
	}
	fullQuery.ModInfo.Type = serverModName
	fullQuery.ModInfo.ModList = pluginList
}

// packagePlayerSection parses and packages the player section into fullQuery.
func packagePlayerSection(playerSection []byte, fullQuery *FullQueryResponse) {
	if len(playerSection) < 4 {
		return
	}

	playerList := []string{}
	playerString := []byte{}

	for _, currentByte := range playerSection {
		// playerString has terminated.
		if currentByte == 0 {
			// Player section has terminated.
			if len(playerString) == 0 {
				break
			}

			playerList = append(playerList, string(playerString))
			playerString = []byte{}
		} else {
			playerString = append(playerString, currentByte)
		}
	}
	fullQuery.Players.PlayerList = playerList
}