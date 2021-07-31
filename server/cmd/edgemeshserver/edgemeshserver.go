package main

import (
	"os"

	"github.com/kubeedge/edgemesh/server/cmd/edgemeshserver/app"
	"k8s.io/component-base/logs"
)

func main() {
	command := app.NewEdgeMeshServerCommand()
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}