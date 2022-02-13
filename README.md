# mcstatusgo
`mcstatusgo` is a pure Go Minecraft service status checker for Java edition Minecraft servers.

`mcstatusgo` supports requesting information through the `status` and `query` protocols. 

## Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/millkhan/mcstatusgo"
)

func main() {
	// Experiment with both the initialTimeout and ioTimeout values to see what works best.
	initialTimeout := time.Second * 10
	ioTimeout := time.Second * 5

	status, err := mcstatusgo.Status("mc.piglin.org", 25565, initialTimeout, ioTimeout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Player count: %d\n", status.Players.Max)

	basicQuery, err := mcstatusgo.BasicQuery("mc.piglin.org", 25565, initialTimeout, ioTimeout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Map Name: %s\n", basicQuery.MapName)

	fullQuery, err := mcstatusgo.FullQuery("mc.piglin.org", 25565, initialTimeout, ioTimeout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server version: %s\n", fullQuery.Version.Name)
}
```

## Documentation

https://pkg.go.dev/github.com/millkhan/mcstatusgo

## Installation

mcstatusgo can be installed easily using the following command:
```bash
go get github.com/millkhan/mcstatusgo
```

## License

`mcstatusgo` is licensed under the MIT License.
Check [LICENSE](LICENSE) for more information.