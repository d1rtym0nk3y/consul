package dataplane

import (
	"context"
	"testing"

	"github.com/hashicorp/consul/agent/grpc/public/testutils"
	"github.com/hashicorp/consul/proto-public/pbdataplane"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func testClient(t *testing.T, server *Server) pbdataplane.DataplaneServiceClient {
	t.Helper()

	addr := testutils.RunTestServer(t, server)

	conn, err := grpc.DialContext(context.Background(), addr.String(), grpc.WithInsecure())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	return pbdataplane.NewDataplaneServiceClient(conn)
}
