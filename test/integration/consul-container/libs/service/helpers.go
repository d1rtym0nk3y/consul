package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/go-cleanhttp"

	"github.com/hashicorp/consul/api"

	libcluster "github.com/hashicorp/consul/test/integration/consul-container/libs/cluster"
	"github.com/hashicorp/consul/test/integration/consul-container/libs/utils"
)

const (
	StaticServerServiceName = "static-server"
	StaticClientServiceName = "static-client"
)

type ServiceOpts struct {
	Name string
	ID   string
	Meta map[string]string
}

func CreateAndRegisterStaticServerAndSidecar(node libcluster.Agent, serviceOpts *ServiceOpts) (Service, Service, error) {
	// Do some trickery to ensure that partial completion is correctly torn
	// down, but successful execution is not.
	var deferClean utils.ResettableDefer
	defer deferClean.Execute()

	// Register the static-server service and sidecar first to prevent race with sidecar
	// trying to get xDS before it's ready
	req := &api.AgentServiceRegistration{
		Name: serviceOpts.Name,
		ID:   serviceOpts.ID,
		Port: 8080,
		Connect: &api.AgentServiceConnect{
			SidecarService: &api.AgentServiceRegistration{
				Proxy: &api.AgentServiceConnectProxyConfig{},
			},
		},
		Check: &api.AgentServiceCheck{
			Name:     "Static Server Listening",
			TCP:      fmt.Sprintf("127.0.0.1:%d", 8080),
			Interval: "10s",
			Status:   api.HealthPassing,
		},
		Meta: serviceOpts.Meta,
	}

	if err := node.GetClient().Agent().ServiceRegister(req); err != nil {
		return nil, nil, err
	}

	// Create a service and proxy instance
	serverService, err := NewExampleService(context.Background(), StaticServerServiceName, 8080, 8079, node)
	if err != nil {
		return nil, nil, err
	}
	deferClean.Add(func() {
		_ = serverService.Terminate()
	})

	serverConnectProxy, err := NewConnectService(context.Background(), fmt.Sprintf("%s-sidecar", StaticServerServiceName), serviceOpts.ID, 8080, node) // bindPort not used
	if err != nil {
		return nil, nil, err
	}
	deferClean.Add(func() {
		_ = serverConnectProxy.Terminate()
	})

	// disable cleanup functions now that we have an object with a Terminate() function
	deferClean.Reset()

	return serverService, serverConnectProxy, nil
}

func CreateAndRegisterStaticClientSidecar(
	node libcluster.Agent,
	peerName string,
	localMeshGateway bool,
) (*ConnectContainer, error) {
	// Do some trickery to ensure that partial completion is correctly torn
	// down, but successful execution is not.
	var deferClean utils.ResettableDefer
	defer deferClean.Execute()

	mgwMode := api.MeshGatewayModeRemote
	if localMeshGateway {
		mgwMode = api.MeshGatewayModeLocal
	}

	// Register the static-client service and sidecar first to prevent race with sidecar
	// trying to get xDS before it's ready
	req := &api.AgentServiceRegistration{
		Name: StaticClientServiceName,
		Port: 8080,
		Connect: &api.AgentServiceConnect{
			SidecarService: &api.AgentServiceRegistration{
				Proxy: &api.AgentServiceConnectProxyConfig{
					Upstreams: []api.Upstream{{
						DestinationName:  StaticServerServiceName,
						DestinationPeer:  peerName,
						LocalBindAddress: "0.0.0.0",
						LocalBindPort:    libcluster.ServiceUpstreamLocalBindPort,
						MeshGateway: api.MeshGatewayConfig{
							Mode: mgwMode,
						},
					}},
				},
			},
		},
	}

	if err := node.GetClient().Agent().ServiceRegister(req); err != nil {
		return nil, err
	}

	// Create a service and proxy instance
	clientConnectProxy, err := NewConnectService(context.Background(), fmt.Sprintf("%s-sidecar", StaticClientServiceName), StaticClientServiceName, libcluster.ServiceUpstreamLocalBindPort, node)
	if err != nil {
		return nil, err
	}
	deferClean.Add(func() {
		_ = clientConnectProxy.Terminate()
	})

	// disable cleanup functions now that we have an object with a Terminate() function
	deferClean.Reset()

	return clientConnectProxy, nil
}

func GetEnvoyConfigDump(port int, filter string) (string, error) {
	client := cleanhttp.DefaultClient()
	url := fmt.Sprintf("http://localhost:%d/config_dump?%s", port, filter)

	res, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func ApplyServiceResolver(cluster *libcluster.Cluster) error {
	client := cluster.Agents[0].GetClient()

	// Register service resolver
	serviceResolver := &api.ServiceResolverConfigEntry{
		Kind:          api.ServiceResolver,
		Name:          StaticServerServiceName,
		DefaultSubset: "v2",
		Subsets: map[string]api.ServiceResolverSubset{
			"v1": {
				Filter: "Service.Meta.version == v1",
			},
			"v2": {
				Filter: "Service.Meta.version == v2",
			},
		},
	}

	ok, _, err := client.ConfigEntries().Set(serviceResolver, nil)
	if err != nil || !ok {
		return fmt.Errorf("error writing service-resolver %w", err)
	}
	return nil
}

// GetSidecarCertificate perform a GET to /certs endpoint and returns response body and any error
func GetSidecarCertificate(port int) (string, error) {
	client := cleanhttp.DefaultClient()
	url := fmt.Sprintf("http://localhost:%d/certs", port)

	res, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// sanitizeResult takes the value returned from config_dump json and cleans it up to remove special characters
// e.g public_listener:0.0.0.0:21001 envoy.filters.network.rbac,envoy.filters.network.tcp_proxy
// returns [envoy.filters.network.rbac envoy.filters.network.tcp_proxy]
func SanitizeResult(s string) []string {
	result := strings.Split(strings.ReplaceAll(s, `,`, " "), " ")
	return append(result[:0], result[1:]...)
}
