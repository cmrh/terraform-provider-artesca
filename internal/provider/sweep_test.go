package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	resource.AddTestSweepers("artesca_replication", &resource.Sweeper{
		Name: "artesca_replication",
		F:    sweepReplication,
	})
	resource.AddTestSweepers("artesca_endpoint", &resource.Sweeper{
		Name:         "artesca_endpoint",
		F:            sweepEndpoints,
		Dependencies: []string{"artesca_replication"},
	})
	resource.AddTestSweepers("artesca_location", &resource.Sweeper{
		Name:         "artesca_location",
		F:            sweepLocations,
		Dependencies: []string{"artesca_endpoint", "artesca_replication"},
	})
	resource.AddTestSweepers("artesca_account", &resource.Sweeper{
		Name:         "artesca_account",
		F:            sweepAccounts,
		Dependencies: []string{"artesca_location"},
	})
}

func sweepAccounts(_ string) error {
	mgmtClient, err := testAccManagementClient()
	if err != nil {
		return nil
	}

	ctx := context.Background()
	overlay, err := mgmtClient.GetOverlay(ctx)
	if err != nil {
		return err
	}

	for _, user := range overlay.Users {
		name := user.AccountName
		if name == "" {
			name = user.UserName
		}
		if strings.HasPrefix(name, "tf-acc") {
			_ = mgmtClient.DeleteAccount(ctx, name)
		}
	}
	return nil
}

func sweepLocations(_ string) error {
	mgmtClient, err := testAccManagementClient()
	if err != nil {
		return nil
	}

	ctx := context.Background()
	overlay, err := mgmtClient.GetOverlay(ctx)
	if err != nil {
		return err
	}

	for name, loc := range overlay.Locations {
		if loc.IsBuiltin {
			continue
		}
		if strings.HasPrefix(name, "tf-acc") {
			_ = mgmtClient.DeleteLocation(ctx, name)
		}
	}
	return nil
}

func sweepEndpoints(_ string) error {
	mgmtClient, err := testAccManagementClient()
	if err != nil {
		return nil
	}

	ctx := context.Background()
	overlay, err := mgmtClient.GetOverlay(ctx)
	if err != nil {
		return err
	}

	for _, ep := range overlay.Endpoints {
		if ep.IsBuiltin {
			continue
		}
		if strings.HasPrefix(ep.Hostname, "tf-acc") {
			_ = mgmtClient.DeleteEndpoint(ctx, ep.Hostname)
		}
	}
	return nil
}

func sweepReplication(_ string) error {
	mgmtClient, err := testAccManagementClient()
	if err != nil {
		return nil
	}

	ctx := context.Background()
	overlay, err := mgmtClient.GetOverlay(ctx)
	if err != nil {
		return err
	}

	for _, rs := range overlay.ReplicationStreams {
		if strings.HasPrefix(rs.Name, "tf-acc") {
			_ = mgmtClient.DeleteReplicationStream(ctx, rs.StreamID)
		}
	}
	return nil
}
