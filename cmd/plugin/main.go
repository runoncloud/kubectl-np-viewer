package main

import (
	"github.com/runoncloud/kubectl-np-viewer/cmd/plugin/cli"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // required for cloud providers
)

func main() {
	cli.InitAndExecute()
}
