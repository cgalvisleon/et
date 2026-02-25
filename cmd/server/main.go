package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/et/utility"
)

func main() {
	port := envar.SetIntByArg("-port", "PORT", 1377)

	node := tcp.NewNode(port)
	node.AddNode("Cesars-MacBook-Pro.local:1377")
	node.AddNode("Cesars-MacBook-Pro.local:1378")
	node.AddNode("Cesars-MacBook-Pro.local:1379")
	err := node.Start()
	if err != nil {
		logs.Panic(err)
	}

	utility.AppWait()
}
