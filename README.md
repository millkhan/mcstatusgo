# mcstatusgo
`mcstatusgo` is a pure Go Minecraft service status checker for Java Edition Minecraft servers.

`mcstatusgo` supports requesting information through four main functions: `status`, `ping`, `basic query`, and `full query`.

## Usage

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
	fmt.Printf("Player count: %d\n", status.Players.Max)

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
	fmt.Printf("Map Name: %s\n", basicQuery.MapName)

	// https://wiki.vg/Query#Full_stat
	fullQuery, err := mcstatusgo.FullQuery("mc.piglin.org", 25565, initialTimeout, ioTimeout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server version: %s\n", fullQuery.Version.Name)
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