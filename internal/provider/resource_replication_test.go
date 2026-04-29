package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccReplication_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rSrcBkt := randomName("tf-acc-sbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_replication.test", "name", "tf-acc-overlay-repl"),
					resource.TestCheckResourceAttr("artesca_replication.test", "version", "1"),
					resource.TestCheckResourceAttr("artesca_replication.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("artesca_replication.test", "stream_id"),
					resource.TestCheckResourceAttr("artesca_replication.test", "source.bucket_name", rSrcBkt),
					resource.TestCheckResourceAttr("artesca_replication.test", "source.prefix", ""),
				),
			},
		},
	})
}

func TestAccReplication_update(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rSrcBkt := randomName("tf-acc-sbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_replication.test", "enabled", "true"),
				),
			},
			{
				Config: testAccReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_replication.test", "enabled", "false"),
				),
			},
		},
	})
}

func TestAccReplication_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rSrcBkt := randomName("tf-acc-sbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, true),
			},
			{
				ResourceName:                         "artesca_replication.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccReplicationImportStateId("artesca_replication.test"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "stream_id",
			},
		},
	})
}

func testAccReplicationImportStateId(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found", resourceName)
		}
		streamID := rs.Primary.Attributes["stream_id"]
		if streamID == "" {
			return "", fmt.Errorf("stream_id not set")
		}
		return streamID, nil
	}
}

func testAccReplicationConfig(acctName, srcLocName, dstLocName, srcBktName string, enabled bool) string {
	return testAccAccountConfig(acctName) +
		testAccLocationSourceConfig(srcLocName) +
		testAccLocationDestConfig(dstLocName) +
		testAccBucketConfig("source", srcBktName, "artesca_location.source.name", true) +
		fmt.Sprintf(`
resource "artesca_replication" "test" {
  name    = "tf-acc-overlay-repl"
  enabled = %t

  source {
    bucket_name = artesca_bucket.source.name
    prefix      = ""
  }

  destination {
    locations {
      name = artesca_location.dest.name
    }
  }
}
`, enabled)
}
