package registration

import (
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

var dns1035LabelFmt = "[a-z]([-a-z0-9]*[a-z0-9])?"
var dns1035LabelRegexp = regexp.MustCompile(dns1035LabelFmt)

func CleanupForKubernetes(input string) string {
	if len(validation.IsDNS1035Label(input)) > 0 {
		r := dns1035LabelRegexp

		// Invalid domain name, Kubernetes will reject this. Try to modify it to a suitable string
		input = strings.ToLower(input)
		input = strings.ReplaceAll(input, "_", "-")
		validParts := r.FindAllString(input, -1)
		return strings.Join(validParts, "")
	}

	return input
}
