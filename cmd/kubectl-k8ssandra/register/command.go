package register

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var RegisterClusterCmd = &cobra.Command{
	Use:   "register [flags]",
	Short: "register a data plane into the control plane.",
	Long:  `register creates a ServiceAccount on a source cluster, copies its credentials and then creates a secret containing them on the destination cluster. It then also creates a ClientConfig on the destination cluster to reference the secret.`,
	Run:   entrypoint,
}

func Init(cmd *cobra.Command, streams genericclioptions.IOStreams) {
	RegisterClusterCmd.Flags().String("source-kubeconfig",
		"",
		"path to source cluster's kubeconfig file - defaults to KUBECONFIG then ~/.kube/config")
	RegisterClusterCmd.Flags().String("dest-kubeconfig",
		"",
		"path to destination cluster's kubeconfig file - defaults to KUBECONFIG then ~/.kube/config")
	RegisterClusterCmd.Flags().String("source-context", "", "context name for source cluster")
	RegisterClusterCmd.Flags().String("dest-context", "", "context name for destination cluster")
	RegisterClusterCmd.Flags().String("source-namespace", "", "namespace containing service account for source cluster")
	RegisterClusterCmd.Flags().String("dest-namespace", "", "namespace where secret and clientConfig will be created on destination cluster")
	RegisterClusterCmd.Flags().String("serviceaccount-name", "k8ssandra-operator", "serviceaccount name for destination cluster")
	RegisterClusterCmd.Flags().String("destination-name", "remote-k8ssandra-operator", "name for remote clientConfig and secret on destination cluster")
	cmd.AddCommand(RegisterClusterCmd)
}

func entrypoint(cmd *cobra.Command, args []string) {
	executor := NewRegistrationExecutorFromRegisterClusterCmd(*cmd)
	for i := 0; i < 30; i++ {
		res := executor.RegisterCluster()
		switch {
		case res.IsError():
			fmt.Println("Registration continuing", res.GetError())
			continue
		case res.Completed():
			fmt.Println("Registration completed successfully")
			return
		case res.IsRequeue():
			fmt.Println("Registration continues")
			continue
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
