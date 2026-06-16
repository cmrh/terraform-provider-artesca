package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccWorkflowReplication_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rSrcBkt := randomName("tf-acc-sbkt")
	rDstBkt := randomName("tf-acc-dbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, 1, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.test", "name", "tf-acc-repl"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.test", "version", "1"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.test", "enabled", "true"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.test", "bucket_name", rSrcBkt),
					resource.TestCheckResourceAttrSet("artesca_bucket_workflow_replication.test", "workflow_id"),
					resource.TestCheckResourceAttrSet("artesca_bucket_workflow_replication.test", "instance_id"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.test", "source.bucket_name", rSrcBkt),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.test", "source.prefix", ""),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.test", "destination.bucket_name", rDstBkt),
				),
			},
		},
	})
}

func TestAccWorkflowReplication_update(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rSrcBkt := randomName("tf-acc-sbkt")
	rDstBkt := randomName("tf-acc-dbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, 1, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.test", "enabled", "true"),
				),
			},
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, 1, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.test", "enabled", "false"),
				),
			},
		},
	})
}

func TestAccWorkflowReplication_validateConfigRejectLocation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "artesca_bucket_workflow_replication" "test" {
  account_id  = "fake-account-id"
  bucket_name = "fake-bucket"
  name        = "test-repl"
  version     = 1
  enabled     = true

  source {
    bucket_name = "fake-bucket"
    prefix      = ""
  }

  destination {
    location = "some-location"
  }
}
`,
				ExpectError: regexp.MustCompile(`(?s)does not support.*destination\.location`),
			},
		},
	})
}

func TestAccWorkflowReplication_validateConfigRejectLocations(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "artesca_bucket_workflow_replication" "test" {
  account_id  = "fake-account-id"
  bucket_name = "fake-bucket"
  name        = "test-repl"
  version     = 1
  enabled     = true

  source {
    bucket_name = "fake-bucket"
    prefix      = ""
  }

  destination {
    locations {
      name = "some-location"
    }
  }
}
`,
				ExpectError: regexp.MustCompile(`(?s)does not support.*destination\.locations`),
			},
		},
	})
}

func TestAccWorkflowReplication_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rSrcBkt := randomName("tf-acc-sbkt")
	rDstBkt := randomName("tf-acc-dbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, 1, true)},
			{
				ResourceName:      "artesca_bucket_workflow_replication.test",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateWorkflowReplication("artesca_bucket_workflow_replication.test"),
				ImportStateVerify: true,
				// workflow_search returns null name/version for replication workflows
				// (issue #37), and the data we round-trip preserves null. ignore those
				// two until the upstream fix lands.
				ImportStateVerifyIdentifierAttribute: "workflow_id",
				ImportStateVerifyIgnore:              []string{"name", "version"},
			},
		},
	})
}

func testAccImportStateWorkflowReplication(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found", resourceName)
		}
		return fmt.Sprintf("%s/%s/%s",
			rs.Primary.Attributes["account_id"],
			rs.Primary.Attributes["bucket_name"],
			rs.Primary.Attributes["workflow_id"],
		), nil
	}
}

func testAccWorkflowReplicationConfig(acctName, srcLocName, dstLocName, srcBktName, dstBktName string, version int, enabled bool) string {
	return testAccAccountConfig(acctName) +
		testAccLocationSourceConfig(srcLocName) +
		testAccLocationDestConfig(dstLocName) +
		testAccBucketConfig("source", srcBktName, "artesca_location.source.name", true) +
		testAccBucketConfig("dest", dstBktName, "artesca_location.dest.name", true) +
		fmt.Sprintf(`
resource "artesca_bucket_workflow_replication" "test" {
  account_id  = artesca_account.test.id
  bucket_name = artesca_bucket.source.name
  name        = "tf-acc-repl"
  version     = %d
  enabled     = %t

  source {
    bucket_name = artesca_bucket.source.name
    prefix      = ""
  }

  destination {
    bucket_name = artesca_bucket.dest.name
  }
}
`, version, enabled)
}
