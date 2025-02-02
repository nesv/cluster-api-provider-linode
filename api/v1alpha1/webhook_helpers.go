package v1alpha1

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"time"

	"github.com/linode/linodego"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/linode/cluster-api-provider-linode/clients"
)

const (
	// defaultWebhookTimeout is the default timeout for an admission request
	defaultWebhookTimeout = time.Second * 10
	// defaultClientTimeout is the default timeout for a client Linode API call
	defaultClientTimeout = time.Second * 10
)

var (
	// defaultLinodeClient is an unauthenticated Linode client
	defaultLinodeClient = linodego.NewClient(&http.Client{Timeout: defaultClientTimeout})
)

func validateRegion(ctx context.Context, client LinodeClient, id string, path *field.Path, capabilities ...string) *field.Error {
	region, err := client.GetRegion(ctx, id)
	if err != nil {
		return field.NotFound(path, id)
	}

	for _, capability := range capabilities {
		if !slices.Contains(region.Capabilities, capability) {
			return field.Invalid(path, id, fmt.Sprintf("no capability: %s", capability))
		}
	}

	return nil
}

func validateLinodeType(ctx context.Context, client LinodeClient, id string, path *field.Path) (*linodego.LinodeType, *field.Error) {
	plan, err := client.GetType(ctx, id)
	if err != nil {
		return nil, field.NotFound(path, id)
	}

	return plan, nil
}

// validateObjectStorageCluster validates an Object Storage deployment's cluster ID via the following rules:
//   - The cluster ID is in the form: REGION_ID-ORDINAL.
//   - The region has Object Storage support.
//
// NOTE: This implementation intended to bypass the authentication requirement for the [Clusters List] and [Cluster
// View] endpoints in the Linode API, thereby reusing a [github.com/linode/linodego.Client] (and its caching if enabled)
// across many admission requests.
//
// [Clusters List]: https://www.linode.com/docs/api/object-storage/#clusters-list
// [Cluster View]: https://www.linode.com/docs/api/object-storage/#cluster-view
func validateObjectStorageCluster(ctx context.Context, client LinodeClient, id string, path *field.Path) *field.Error {
	//nolint:gocritic // prefer no escapes
	cexp := regexp.MustCompile("^(([[:lower:]]+-)*[[:lower:]]+)-[[:digit:]]+$")
	if !cexp.MatchString(id) {
		return field.Invalid(path, id, "must be in form: region_id-ordinal")
	}

	region := cexp.FindStringSubmatch(id)[1]
	return validateRegion(ctx, client, region, path, LinodeObjectStorageCapability)
}
