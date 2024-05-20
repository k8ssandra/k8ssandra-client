package register

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var RegisterClusterCmd = &cobra.Command{
	Use:   "register [flags]",
	Short: "register a data plane into the control plane.",
	Long:  `register creates a ServiceAccount on a source cluster, copies its credentials and then creates a secret containing them on the destination cluster. It then also creates a ClientConfig on the destination cluster to reference the secret.`,
	Run:   entrypoint,
}

func SetupRegisterClusterCmd(cmd *cobra.Command, streams genericclioptions.IOStreams) {
	RegisterClusterCmd.Flags().String("source-kubeconfig",
		"",
		"path to source cluster's kubeconfig file - defaults to KUBECONFIG then ~/.kube/config")
	RegisterClusterCmd.Flags().String("dest-kubeconfig",
		"",
		"path to destination cluster's kubeconfig file - defaults to KUBECONFIG then ~/.kube/config")
	RegisterClusterCmd.Flags().String("source-context", "", "context name for source cluster")
	RegisterClusterCmd.Flags().String("dest-context", "", "context name for destination cluster")
	RegisterClusterCmd.Flags().String("source-namespace", "k8ssandra-operator", "namespace containing service account for source cluster")
	RegisterClusterCmd.Flags().String("dest-namespace", "k8ssandra-operator", "namespace where secret and clientConfig will be created on destination cluster")
	RegisterClusterCmd.Flags().String("serviceaccount-name", "k8ssandra-operator", "serviceaccount name for destination cluster")
	RegisterClusterCmd.Flags().String("destination-name", "remote-k8ssandra-operator", "name for remote clientConfig and secret on destination cluster")

	if err := RegisterClusterCmd.MarkFlagRequired("source-context"); err != nil {
		panic(err)

	}
	if err := RegisterClusterCmd.MarkFlagRequired("dest-context"); err != nil {
		panic(err)
	}
	cmd.AddCommand(RegisterClusterCmd)
}

func entrypoint(cmd *cobra.Command, args []string) {
	executor := NewRegistrationExecutorFromRegisterClusterCmd(*cmd)
	// insert validation that if e.SourceContext == e.DestContext && e.SourceKubeconfig == e.DestKubeconfig {
	for i := 0; i < 30; i++ {
		res := executor.RegisterCluster()
		switch v := res.(type) {
		case RetryableError:
			log.Info("Registration continuing", "msg", v.Error())
			continue
		case nil:
			log.Info("Registration completed successfully")
			return
		case NonRecoverableError:
			panic(fmt.Sprintf("Registration failed: %s", v.Error()))
		}
	}
	fmt.Println("Registration failed - retries exceeded")
}

func NewRegistrationExecutorFromRegisterClusterCmd(cmd cobra.Command) *RegistrationExecutor {
	return &RegistrationExecutor{
		SourceKubeconfig: cmd.Flag("source-kubeconfig").Value.String(),
		DestKubeconfig:   cmd.Flag("dest-kubeconfig").Value.String(),
		SourceContext:    cmd.Flag("source-context").Value.String(),
		DestContext:      cmd.Flag("dest-context").Value.String(),
		SourceNamespace:  cmd.Flag("source-namespace").Value.String(),
		DestNamespace:    cmd.Flag("dest-namespace").Value.String(),
		ServiceAccount:   cmd.Flag("serviceaccount-name").Value.String(),
		Context:          cmd.Context(),
		DestinationName:  cmd.Flag("destination-name").Value.String(),
	}
}
