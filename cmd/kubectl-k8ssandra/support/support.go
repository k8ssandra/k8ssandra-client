package support

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	// "github.com/k8ssandra/k8ssandra-client/pkg/collector"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/collect"
	"github.com/replicatedhq/troubleshoot/pkg/constants"
	"github.com/replicatedhq/troubleshoot/pkg/loader"
	"github.com/replicatedhq/troubleshoot/pkg/supportbundle"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/replicatedhq/troubleshoot/pkg/types"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	supportBundleExample = `
	# Process the config files from cass-operator input
	%[1]s support-bundle [<args>]
	`
)

type supportOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams

	outputDir string
}

func newBundleOptions(streams genericclioptions.IOStreams) *supportOptions {
	return &supportOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewCmd provides a cobra command wrapping newAddOptions
func NewSupportBundleCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newBundleOptions(streams)

	cmd := &cobra.Command{
		Use:     "support-bundle [flags]",
		Short:   "Build support-bundle",
		Example: fmt.Sprintf(supportBundleExample, "kubectl k8ssandra support-bundle"),
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	fl := cmd.Flags()
	fl.StringVar(&o.outputDir, "output", "", "write output files to this directory instead of default")
	o.configFlags.AddFlags(fl)
	return cmd
}

// Complete parses the arguments and necessary flags to options
func (c *supportOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (c *supportOptions) Validate() error {
	return nil
}

// Run processes the input, creates a connection to Kubernetes and processes a secret to add the users
func (c *supportOptions) Run() error {
	restConfig, err := c.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()

	progressChan := make(chan interface{})
	collectorCB := func(c chan interface{}, msg string) { c <- msg }
	createOpts := supportbundle.SupportBundleCreateOpts{
		KubernetesRestConfig:      restConfig,
		FromCLI:                   false,
		ProgressChan:              progressChan,
		CollectorProgressCallback: collectorCB,
	}

	// Here we could load multiple rawSpecs but this prototype will not
	kinds, err := loadSpecs(ctx)
	if err != nil {
		return err
	}

	// Merge all the URLs we downloaded
	mainBundle, additionalRedactors, err := mergeSpecs(kinds)
	if err != nil {
		return err
	}

	// Add new commands to the mainBundle.Spec.Collectors ? And then execute it
	go func() {
		for _ = range progressChan {
		}
	}()

	// This can't be done, because the Spec.Collectors is a slice of  troubleshootv1beta2.Collect and not
	// "github.com/replicatedhq/troubleshoot/pkg/collect".Collector
	// protoCollector := &collector.ProtoCollector{}
	// mainBundle.Spec.Collectors = append(mainBundle.Spec.Collectors, protoCollector)

	response, err := supportbundle.CollectSupportBundleFromSpec(&mainBundle.Spec, additionalRedactors, createOpts)
	if err != nil {
		return err
	}

	close(progressChan)

	fmt.Printf("A support bundle has been created in the current directory named %q\n", response.ArchivePath)
	return nil
}

// TODO This is all prototyping, we'll need airgap support in reality
// This is here because https://github.com/replicatedhq/troubleshoot/blob/d83d8ebfc66c1f399175d9c0618886debb88d48e/internal/specs/specs.go#L75 is private
// And because https://github.com/replicatedhq/troubleshoot/blob/d83d8ebfc66c1f399175d9c0618886debb88d48e/cmd/troubleshoot/cli/run.go#L287 is private

func loadSpecs(ctx context.Context) (*loader.TroubleshootKinds, error) {
	v := "https://kots.io"

	// To download specs from kots.io, we need to set the User-Agent header
	specFromURL, err := downloadFromHttpURL(ctx, v, map[string]string{
		"User-Agent": "Replicated_Troubleshoot/v1beta1",
	})
	if err != nil {
		return nil, err
	}

	// load URL spec first to remove URI key from the spec
	kindsFromURL, err := loader.LoadSpecs(ctx, loader.LoadOptions{
		RawSpec: specFromURL,
	})
	if err != nil {
		return nil, err
	}
	// remove URI key from the spec if any
	for i := range kindsFromURL.SupportBundlesV1Beta2 {
		kindsFromURL.SupportBundlesV1Beta2[i].Spec.Uri = ""
	}

	rawSpecs := []string{}
	kinds, err := loader.LoadSpecs(ctx, loader.LoadOptions{
		RawSpecs: rawSpecs,
	})
	if err != nil {
		return nil, err
	}
	if kindsFromURL != nil {
		kinds.Add(kindsFromURL)
	}

	return kinds, err
}

func mergeSpecs(kinds *loader.TroubleshootKinds) (*troubleshootv1beta2.SupportBundle, *troubleshootv1beta2.Redactor, error) {
	mainBundle := &troubleshootv1beta2.SupportBundle{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "troubleshoot.sh/v1beta2",
			Kind:       "SupportBundle",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "merged-support-bundle-spec",
		},
	}
	for _, sb := range kinds.SupportBundlesV1Beta2 {
		sb := sb
		mainBundle = supportbundle.ConcatSpec(mainBundle, &sb)
	}

	if mainBundle.Spec.Collectors == nil {
		mainBundle.Spec.Collectors = make([]*troubleshootv1beta2.Collect, 0)
	}

	for _, c := range kinds.CollectorsV1Beta2 {
		mainBundle.Spec.Collectors = append(mainBundle.Spec.Collectors, c.Spec.Collectors...)
	}

	if mainBundle.Spec.HostCollectors == nil {
		mainBundle.Spec.HostCollectors = make([]*troubleshootv1beta2.HostCollect, 0)
	}
	for _, hc := range kinds.HostCollectorsV1Beta2 {
		mainBundle.Spec.HostCollectors = append(mainBundle.Spec.HostCollectors, hc.Spec.Collectors...)
	}

	if !(len(mainBundle.Spec.HostCollectors) > 0 && len(mainBundle.Spec.Collectors) == 0) {
		// Always add default collectors unless we only have host collectors
		// We need to add them here so when we --dry-run, these collectors
		// are included. supportbundle.runCollectors duplicates this bit.
		// We'll need to refactor it out later when its clearer what other
		// code depends on this logic e.g KOTS
		mainBundle.Spec.Collectors = collect.EnsureCollectorInList(
			mainBundle.Spec.Collectors,
			troubleshootv1beta2.Collect{ClusterInfo: &troubleshootv1beta2.ClusterInfo{}},
		)
		mainBundle.Spec.Collectors = collect.EnsureCollectorInList(
			mainBundle.Spec.Collectors,
			troubleshootv1beta2.Collect{ClusterResources: &troubleshootv1beta2.ClusterResources{}},
		)
	}

	additionalRedactors := &troubleshootv1beta2.Redactor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "troubleshoot.sh/v1beta2",
			Kind:       "Redactor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "merged-redactors-spec",
		},
	}
	if additionalRedactors.Spec.Redactors == nil {
		additionalRedactors.Spec.Redactors = make([]*troubleshootv1beta2.Redact, 0)
	}
	for _, r := range kinds.RedactorsV1Beta2 {
		additionalRedactors.Spec.Redactors = append(additionalRedactors.Spec.Redactors, r.Spec.Redactors...)
	}

	return mainBundle, additionalRedactors, nil
}

// This method isn't exposed in troubleshoot pkg
func downloadFromHttpURL(ctx context.Context, url string, headers map[string]string) (string, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		// exit code: should this be catch all or spec issues...?
		return "", types.NewExitCodeError(constants.EXIT_CODE_CATCH_ALL, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		// exit code: should this be catch all or spec issues...?
		return "", types.NewExitCodeError(constants.EXIT_CODE_CATCH_ALL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", types.NewExitCodeError(constants.EXIT_CODE_SPEC_ISSUES, err)
	}
	return string(body), nil
}
