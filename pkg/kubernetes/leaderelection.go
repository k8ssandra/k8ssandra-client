package kubernetes

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	coordinationv1client "k8s.io/client-go/kubernetes/typed/coordination/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func WaitForLock(ctx context.Context, namespace string, lockName string, client *NamespacedClient) error {
	lock, err := NewResourceLock(namespace, lockName, client.Config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Gain leader election and then proceed
	go RunLeaderElection(ctx, wg, lock)
	wg.Wait()

	return nil
}

func NewResourceLock(namespace, lockName string, restConfig *rest.Config) (resourcelock.Interface, error) {
	// Leader id, needs to be unique
	id, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	padder, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	id = id + "_" + padder.String()

	// Construct clients for leader election
	rest.AddUserAgent(restConfig, "leader-election")
	corev1Client, err := corev1client.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	coordinationClient, err := coordinationv1client.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return resourcelock.New(resourcelock.LeasesResourceLock,
		namespace,
		lockName,
		corev1Client,
		coordinationClient,
		resourcelock.ResourceLockConfig{
			Identity: id,
		})
}

func RunLeaderElection(ctx context.Context, wg *sync.WaitGroup, lock resourcelock.Interface) {
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   10 * time.Second,
		RenewDeadline:   5 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(c context.Context) {
				// Indicate we've acquired the lock..
				wg.Done()
			},
			OnStoppedLeading: func() {
				// TODO This is fatal to our usage..
			},
		},
	})
}
