package tasks

import (
	"context"
	"fmt"
	"time"

	controlapi "github.com/k8ssandra/cass-operator/apis/control/v1alpha1"
	"github.com/k8ssandra/k8ssandra-client/pkg/util"
	"k8s.io/apimachinery/pkg/types"
	waitutil "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WaitForCompletion(ctx context.Context, kubeClient client.Client, task *controlapi.CassandraTask) error {
	taskKey := types.NamespacedName{Name: task.Name, Namespace: task.Namespace}
	return WaitForCompletionKey(ctx, kubeClient, taskKey)
}

func WaitForCompletionKey(ctx context.Context, kubeClient client.Client, taskKey types.NamespacedName) error {
	err := waitutil.PollImmediate(5*time.Second, 10*time.Minute, func() (done bool, err error) {
		task := &controlapi.CassandraTask{}
		if err := kubeClient.Get(ctx, taskKey, task); err != nil {
			return false, err
		}

		return task.Status.CompletionTime != nil, nil
	})

	return err
}

func createName(first, second string) string {
	staticPart := fmt.Sprintf("%s-%s", first, second)
	dynPart := fmt.Sprintf("%d%s", time.Now().UTC().Unix(), util.RandomKubeCompatibleText(8))
	if len(staticPart) > 45 {
		staticPart = staticPart[:45]
	}

	return fmt.Sprintf("%s%s", staticPart, dynPart)
}
