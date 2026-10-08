package kubernetes

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func LoadClientConfig(cfgFile, contextName string) (*rest.Config, error) {
	config, err := clientcmd.LoadFromFile(cfgFile)
	if err != nil {
		return nil, err
	}

	if contextName != "" {
		if _, found := config.Contexts[contextName]; !found {
			return nil, fmt.Errorf("context %s not found in kubeconfig file", contextName)
		}
	}
	config.CurrentContext = contextName

	b, err := clientcmd.Write(*config)
	if err != nil {
		return nil, err
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(b)
	if err != nil {
		return nil, err
	}

	return clientConfig.ClientConfig()
}
