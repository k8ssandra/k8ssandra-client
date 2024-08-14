package collector

import (
	"bytes"
	"context"

	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/collect"
	"k8s.io/client-go/rest"
)

var _ collect.Collector = &ProtoCollector{}

type ProtoCollector struct {
	troubleshootv1beta2.CollectorMeta `json:",inline" yaml:",inline"`
	BundlePath                        string
}

func (p *ProtoCollector) Title() string {
	return "proto"
}

func (p *ProtoCollector) IsExcluded() (bool, error) {
	return false, nil
}

func (p *ProtoCollector) GetRBACErrors() []error {
	return nil
}

func (p *ProtoCollector) HasRBACErrors() bool {
	return false
}

func (p *ProtoCollector) CheckRBAC(ctx context.Context, c collect.Collector, collector *troubleshootv1beta2.Collect, clientConfig *rest.Config, namespace string) error {
	return nil
}

func (p *ProtoCollector) Collect(progressChan chan<- interface{}) (collect.CollectorResult, error) {
	res := collect.NewResult()

	// Generic note, but why does this collector need to know the BundlePath? Shouldn't upper stage have enough
	// information to parse the CollectorResult to that correct location?
	if err := res.SaveResult(p.BundlePath, "proto.txt", bytes.NewBufferString("something to show")); err != nil {
		return nil, err
	}
	return res, nil
}
