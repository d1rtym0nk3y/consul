package l7trafficmanagement

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	libassert "github.com/hashicorp/consul/test/integration/consul-container/libs/assert"
	libcluster "github.com/hashicorp/consul/test/integration/consul-container/libs/cluster"
	libservice "github.com/hashicorp/consul/test/integration/consul-container/libs/service"
	"github.com/hashicorp/consul/test/integration/consul-container/libs/utils"
	"github.com/stretchr/testify/require"
)

// TestTrafficManagement_Upgrade Summary
// This test starts up 2 servers and 1 client in the same datacenter.
//
// Steps:
//   - Create a single agent cluster.
//   - Create one static-server and 2 subsets and 1 client and sidecar, then register them with Consul
//   - Validate static-server and 2 subsets are and proxy admin endpoint is healthy
//   - Validate static servers proxy listeners should be up and have right certs
func TestTrafficManagement_SetupServerAndClientWithSubsets(t *testing.T) {
	t.Parallel()

	type testcase struct {
		oldversion    string
		targetVersion string
	}
	tcs := []testcase{
		{
			oldversion:    "1.14",
			targetVersion: utils.TargetVersion,
		},
	}

	run := func(t *testing.T, tc testcase) {
		cluster := createCluster(t, tc.oldversion)
		err := libservice.ApplyServiceResolver(cluster)
		require.NoError(t, err)
		_, _, serverServiceV2, clientService := createService(t, cluster)

		_, port := clientService.GetAddr()
		_, adminPort := clientService.GetAdminAddr()

		//get serverport
		_, serverAdminPort := serverServiceV2.GetAdminAddr()
		fmt.Println(port, adminPort, serverAdminPort)
		libassert.AssertUpstreamEndpointStatus(t, adminPort, "v2.static-server.default", "HEALTHY", 1)
		// libassert.HTTPServiceEchoes(t, "localhost", clientPort, "")

		// // Upgrade cluster and begin service validation
		// require.NoError(t, cluster.StandardUpgrade(t, context.Background(), tc.targetVersion))

		// // POST upgrade validation
		// // validate static-server, static-server-v1, static-server-v2 and static-client
		// // proxy admin are up
		// libassert.AssertServiceProxyAdminStatus(t, "localhost", clientAdminPort)
		// libassert.AssertServiceProxyAdminStatus(t, "localhost", adminPort)
		// libassert.AssertServiceProxyAdminStatus(t, "localhost", adminPortV1)
		// libassert.AssertServiceProxyAdminStatus(t, "localhost", adminPortV2)

		// // certs are valid
		// libassert.AssertEnvoyPresentsCertURI(t, clientAdminPort, "static-client", 3)
		// libassert.AssertEnvoyPresentsCertURI(t, adminPort, "static-server", 2)
		// libassert.AssertEnvoyPresentsCertURI(t, adminPortV1, "static-server", 2)
		// libassert.AssertEnvoyPresentsCertURI(t, adminPortV2, "static-server", 2)

		time.Sleep(900 * time.Second)

		// TO-DO: validate traffic management
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("upgrade from %s to %s", tc.oldversion, tc.targetVersion),
			func(t *testing.T) {
				run(t, tc)
			})
	}
}

func createCluster(t *testing.T, version string) *libcluster.Cluster {
	opts := libcluster.BuildOptions{
		InjectAutoEncryption:   true,
		InjectGossipEncryption: true,
		AllowHTTPAnyway:        true,
		ConsulVersion:          version,
	}
	ctx := libcluster.NewBuildContext(t, opts)

	conf := libcluster.NewConfigBuilder(ctx).
		ToAgentConfig(t)
	t.Logf("Cluster config:\n%s", conf.JSON)

	configs := []libcluster.Config{*conf}

	cluster, err := libcluster.New(t, configs)
	require.NoError(t, err)

	node := cluster.Agents[0]
	client := node.GetClient()

	libcluster.WaitForLeader(t, cluster, client)
	libcluster.WaitForMembers(t, client, 1)

	// Apply HTTP Proxy Settings
	proxyDefault := &api.ProxyConfigEntry{
		Kind: api.ProxyDefaults,
		Name: api.ProxyConfigGlobal,
		Config: map[string]any{
			"protocol": "http",
		},
	}
	set, _, err := cluster.Agents[0].GetClient().ConfigEntries().Set(proxyDefault, nil)
	require.NoError(t, err)
	require.True(t, set)

	return cluster
}

// create 3 servers and 1 client
func createService(t *testing.T, cluster *libcluster.Cluster) (libservice.Service, libservice.Service, libservice.Service, libservice.Service) {
	node := cluster.Agents[0]
	client := node.GetClient()

	serviceOptsV2 := &libservice.ServiceOpts{
		Name: libservice.StaticServerServiceName,
		ID:   "static-server-v2",
		Meta: map[string]string{"version": "v2"},
	}
	_, serverServiceV2, err := libservice.CreateAndRegisterStaticServerAndSidecar(node, serviceOptsV2)
	libassert.CatalogServiceExists(t, client, "static-server")
	require.NoError(t, err)

	// Create a client proxy instance with the server as an upstream
	clientService, err := libservice.CreateAndRegisterStaticClientSidecar(node, "", false)
	require.NoError(t, err)
	libassert.CatalogServiceExists(t, client, fmt.Sprintf("%s-sidecar-proxy", libservice.StaticClientServiceName))

	return nil, nil, serverServiceV2, clientService
}
