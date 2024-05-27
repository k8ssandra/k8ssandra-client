package registration

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TokenToKubeconfig(s corev1.Secret, server string) (clientcmdapi.Config, error) {
	caData, foundCa := s.Data["ca.crt"]
	tokenData, foundToken := s.Data["token"]
	if !foundCa || !foundToken {
		return clientcmdapi.Config{}, errors.New("missing required data in secret")
	}

	return clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"cluster": {
				Server:                   server,
				CertificateAuthorityData: caData,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"cluster": {
				Token: string(tokenData),
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"cluster": {
				Cluster:  "cluster",
				AuthInfo: "cluster",
			},
		},
		CurrentContext: "cluster",
	}, nil
}
