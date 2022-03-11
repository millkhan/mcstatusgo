# mcstatusgo
`mcstatusgo` is a pure Go Minecraft service status checker for Java Edition Minecraft servers.

`mcstatusgo` supports requesting information through six protocols: `status`, `legacy status`, `beta status`, `ping`, `basic query`, and `full query`.

`status`, `ping`, `basic query`, and `full query` are the most up-to-date protocols.

`legacy status` and `beta status` are older implementations of `status` used in older versions of Minecraft.

## Usage

#### Current Protocols
```go
package main

import (
	"fmt"
	"time"

	"github.com/millkhan/mcstatusgo/v2"
)

func main() {
	// Experiment with both the initialTimeout and ioTimeout values to see what works best.
	initialTimeout := time.Second * 10
	ioTimeout := time.Second * 5

	// https://wiki.vg/Server_List_Ping
	status, err := mcstatusgo.Status("mc.piglin.org", 25565, initialTimeout, ioTimeout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Max player count: %d\n", status.Players.Max)

	// https://wiki.vg/Server_List_Ping#Ping
	ping, err := mcstatusgo.Ping("mc.piglin.org", 25565, initialTimeout, ioTimeout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server latency: %s\n", ping)

	// https://wiki.vg/Query#Basic_stat
	basicQuery, err := mcstatusgo.BasicQuery("mc.piglin.org", 25565, initialTimeout, ioTimeout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Map name: %s\n", basicQuery.MapName)

	// https://wiki.vg/Query#Full_stat
	fullQuery, err := mcstatusgo.FullQuery("mc.piglin.org", 25565, initialTimeout, ioTimeout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server version: %s\n", fullQuery.Version.Name)
}
```

#### Older Protocols
```go
package main

import (
	"fmt"
	"time"

	"github.com/millkhan/mcstatusgo/v2"
)

func main() {
	// Experiment with both the initialTimeout and ioTimeout values to see what works best.
	initialTimeout := time.Second * 10
	ioTimeout := time.Second * 5

	// https://wiki.vg/Server_List_Ping#1.6
	statusLegacy, err := mcstatusgo.StatusLegacy("us.mineplex.com", 25565, initialTimeout, ioTimeout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Max player count: %d\n", statusLegacy.Players.Max)

	// https://wiki.vg/Server_List_Ping#Beta_1.8_to_1.3
	statusBeta, err := mcstatusgo.StatusBeta("us.mineplex.com", 25565, initialTimeout, ioTimeout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Online player count: %d\n", statusBeta.Players.Online)
}
```

## Documentation

https://pkg.go.dev/github.com/millkhan/mcstatusgo/v2

## Installation

mcstatusgo can be installed easily using the following command:
```bash
go get github.com/millkhan/mcstatusgo/v2
```

## License

`mcstatusgo` is licensed under the MIT License.
Check [LICENSE](LICENSE) for more information.
