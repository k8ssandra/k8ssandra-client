package register

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/k8ssandra/k8ssandra-client/pkg/registration"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var RegisterClusterCmd = &cobra.Command{
	Use:   "register [flags]",
	Short: "register a data plane into the control plane.",
	Long:  `register creates a ServiceAccount on a source cluster, copies its credentials and then creates a secret containing them on the destination cluster. It then also creates a ClientConfig on the destination cluster to reference the secret.`,
	RunE:  entrypoint,
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
	RegisterClusterCmd.Flags().String("destination-name", "", "name for remote clientConfig and secret on destination cluster")
	RegisterClusterCmd.Flags().String("oride-src-ip", "", "override source IP for when you need to specify a different IP for the source cluster than is contained in kubeconfig")
	RegisterClusterCmd.Flags().String("oride-src-port", "", "override source port for when you need to specify a different port for the source cluster than is contained in src kubeconfig")

	if err := RegisterClusterCmd.MarkFlagRequired("source-context"); err != nil {
		panic(err)

	}
	if err := RegisterClusterCmd.MarkFlagRequired("dest-context"); err != nil {
		panic(err)
	}
	RegisterClusterCmd.MarkFlagsRequiredTogether("oride-src-ip", "oride-src-port")
	cmd.AddCommand(RegisterClusterCmd)
}

func entrypoint(cmd *cobra.Command, args []string) error {
	executor := NewRegistrationExecutorFromRegisterClusterCmd(*cmd)

	// TODO What is this magic number 30?
	for i := 0; i < 30; i++ {
		if err := executor.RegisterCluster(); err != nil {
			if errors.Is(err, NonRecoverableError{}) {
				log.Error(fmt.Sprintf("Registration failed: %s", err.Error()))
				return err
			}
			log.Info("Registration still in progress", "msg", err.Error())
			continue
		}
		log.Info("Registration completed successfully")
		return nil
	}
	log.Error("Registration failed - retries exceeded")
	return nil
}

func NewRegistrationExecutorFromRegisterClusterCmd(cmd cobra.Command) *RegistrationExecutor {

	destName := cmd.Flag("destination-name").Value.String()
	srcContext := cmd.Flag("source-context").Value.String()
	if destName == "" {
		destName = registration.CleanupForKubernetes(srcContext)
	}
	return &RegistrationExecutor{
		SourceKubeconfig:   cmd.Flag("source-kubeconfig").Value.String(),
		DestKubeconfig:     cmd.Flag("dest-kubeconfig").Value.String(),
		SourceContext:      srcContext,
		DestContext:        cmd.Flag("dest-context").Value.String(),
		SourceNamespace:    cmd.Flag("source-namespace").Value.String(),
		DestNamespace:      cmd.Flag("dest-namespace").Value.String(),
		ServiceAccount:     cmd.Flag("serviceaccount-name").Value.String(),
		OverrideSourceIP:   cmd.Flag("override-src-ip").Value.String(),
		OverrideSourcePort: cmd.Flag("override-src-port").Value.String(),
		Context:            cmd.Context(),
		DestinationName:    destName,
	}
}
