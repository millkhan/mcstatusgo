# mcstatusgo
`mcstatusgo` is a pure Go Minecraft service status checker for Java edition Minecraft servers.

`mcstatusgo` supports requesting information through the `status` and `query` protocol. 

Usage
-----
```go
package main

import (
	"fmt"
	"time"

	"github.com/millkhan/mcstatusgo"
)

func main() {
	status, err := mcstatusgo.Status("mc.hypixel.net", 25565, time.Second*10, time.Second*5)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Hypixel player count: %d\n", status.Players.Max)

	query, err := mcstatusgo.Query("mc.piglin.org", 25565, time.Second*10, time.Second*10)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Piglin server version: %s\n", query.Version.Name)
}
```

## Installation
----------
mcstatusgo can be installed easily using the following command:
```bash
go get github.com/millkhan/mcstatusgo
```
License
-------

`mcstatusgo` is licensed under the MIT License.
Check [LICENSE](LICENSE) for more information.
