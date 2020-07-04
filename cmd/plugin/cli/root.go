package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/runoncloud/kubectl-np/pkg/logger"
	"github.com/runoncloud/kubectl-np/pkg/plugin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "kubectl-np",
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

			//s := spin.New()
			finishedCh := make(chan bool, 1)
			namespaceName := make(chan string, 1)

			go func() {
				lastNamespaceName := ""
				for {
					select {
					case <-finishedCh:
						fmt.Printf("\r")
						return

					case n := <-namespaceName:
						lastNamespaceName = n

					case <-time.After(time.Millisecond * 100):
						if lastNamespaceName == "" {
							//fmt.Printf("\r  \033[36mSearching for namespaces\033[m %s", s.Next())
						} else {
							//fmt.Printf("\r  \033[36mSearching for namespaces\033[m %s (%s)", s.Next(), lastNamespaceName)
						}
					}
				}
			}()

			defer func() {
				finishedCh <- true
			}()

			if err := plugin.RunPlugin(KubernetesConfigFlags, cmd); err != nil {
				return errors.Cause(err)
			}

			log.Info("")

			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	var ingress, egress, allNamespaces bool
	var pod string
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
