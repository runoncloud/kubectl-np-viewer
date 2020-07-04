package cli

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/runoncloud/kubectl-np/pkg/logger"
	"github.com/runoncloud/kubectl-np/pkg/plugin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"strings"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
)

func RootCmd() *cobra.Command {
	var ingress, egress, allNamespaces bool
	var pod string

	cmd := &cobra.Command{
		Use:           "np-viewer",
		Short:         "",
		Long:          `.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log := logger.NewLogger()
			log.Info("")

			finishedCh := make(chan bool, 1)
			go func() {
				for {
					select {
					case <-finishedCh:
						fmt.Printf("\r")
						return
					}
				}
			}()

			defer func() {
				finishedCh <- true
			}()

			if err := plugin.RunPlugin(KubernetesConfigFlags, cmd); err != nil {
				return errors.Cause(err)
			}
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	cmd.Flags().BoolVarP(&ingress, "ingress", "i", false, "Only select ingress type")
	cmd.Flags().BoolVarP(&egress, "egress", "e", false, " Only select egress type")
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, " All namespaces")
	cmd.Flags().StringVarP(&pod, "pod", "p", "", "Rule apply to a specific pod")

	KubernetesConfigFlags = genericclioptions.NewConfigFlags(false)
	KubernetesConfigFlags.AddFlags(cmd.Flags())

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.AutomaticEnv()
}
