package assert

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil/retry"
	libservice "github.com/hashicorp/consul/test/integration/consul-container/libs/service"
)

const (
	defaultHTTPTimeout = 100 * time.Second
	defaultHTTPWait    = defaultWait
)

// CatalogServiceExists verifies the service name exists in the Consul catalog
func CatalogServiceExists(t *testing.T, c *api.Client, svc string) {
	retry.Run(t, func(r *retry.R) {
		services, _, err := c.Catalog().Service(svc, "", nil)
		if err != nil {
			r.Fatal("error reading peering data")
		}
		if len(services) == 0 {
			r.Fatal("did not find catalog entry for ", svc)
		}
	})
}

// HTTPServiceEchoes verifies that a post to the given ip/port combination returns the data
// in the response body. Optional path can be provided to differentiate requests.
func HTTPServiceEchoes(t *testing.T, ip string, port int, path string) {
	const phrase = "hello"

	failer := func() *retry.Timer {
		return &retry.Timer{Timeout: defaultHTTPTimeout, Wait: defaultHTTPWait}
	}

	client := cleanhttp.DefaultClient()
	url := fmt.Sprintf("http://%s:%d", ip, port)

	if path != "" {
		url += "/" + path
	}

	retry.RunWith(failer(), t, func(r *retry.R) {
		t.Logf("making call to %s", url)
		reader := strings.NewReader(phrase)
		res, err := client.Post(url, "text/plain", reader)
		if err != nil {
			r.Fatal("could not make call to service ", url)
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			r.Fatal("could not read response body ", url)
		}

		if !strings.Contains(string(body), phrase) {
			r.Fatal("received an incorrect response ", string(body))
		}
	})
}

// AssertServiceProxyAdminStatus validates all service proxy sidecards are up and running.
func AssertServiceProxyAdminStatus(t *testing.T, ip string, port int) {
	var statusCode int
	failer := func() *retry.Timer {
		return &retry.Timer{Timeout: defaultHTTPTimeout, Wait: defaultHTTPWait}
	}

	client := cleanhttp.DefaultClient()
	url := fmt.Sprintf("http://%s:%d/stats", ip, port)

	retry.RunWith(failer(), t, func(r *retry.R) {
		t.Logf("making call to %s", url)
		res, err := client.Get(url)
		if err != nil {
			r.Fatal("could not make call to service ", url)
		}
		defer res.Body.Close()
		statusCode = res.StatusCode
	})
	assert.Equal(t, http.StatusOK, statusCode, fmt.Sprintf("service cannot be reached %v", statusCode))
}

// ServiceLogContains returns true if the service container has the target string in its logs
func ServiceLogContains(t *testing.T, service libservice.Service, target string) bool {
	logs, err := service.GetLogs()
	require.NoError(t, err)
	return strings.Contains(logs, target)
}
