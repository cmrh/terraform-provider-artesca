package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testAccReplicationResource = "artesca_bucket_workflow_replication.test"

func TestAccWorkflowReplication_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rSrcBkt := randomName("tf-acc-sbkt")
	rDstBkt := randomName("tf-acc-dbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowReplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, true, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccReplicationResource, "enabled", "true"),
					resource.TestCheckResourceAttr(testAccReplicationResource, "bucket_name", rSrcBkt),
					resource.TestCheckResourceAttr(testAccReplicationResource, "destination_bucket_name", rDstBkt),
					resource.TestCheckResourceAttrSet(testAccReplicationResource, "rule_id"),
					testAccCheckReplicationRules(rSrcBkt, 1),
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
		CheckDestroy:             testAccCheckWorkflowReplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, true, ""),
				Check:  resource.TestCheckResourceAttr(testAccReplicationResource, "enabled", "true"),
			},
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, false, "logs/"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccReplicationResource, "enabled", "false"),
					resource.TestCheckResourceAttr(testAccReplicationResource, "filter.object_key_prefix", "logs/"),
				),
			},
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, true, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccReplicationResource, "enabled", "true"),
					resource.TestCheckNoResourceAttr(testAccReplicationResource, "filter.object_key_prefix"),
				),
			},
		},
	})
}

// Two rules on one bucket exercise the read-merge-write path: creating the
// second must keep the first, and removing one must keep the other.
func TestAccWorkflowReplication_multipleRules(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rSrcBkt := randomName("tf-acc-sbkt")
	rDstBkt := randomName("tf-acc-dbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowReplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, true, "a/") +
					testAccWorkflowReplicationExtraRule("b/"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccReplicationResource, "filter.object_key_prefix", "a/"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.second", "filter.object_key_prefix", "b/"),
					testAccCheckReplicationRules(rSrcBkt, 2),
				),
			},
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, true, "a/"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccReplicationResource, "filter.object_key_prefix", "a/"),
					testAccCheckReplicationRules(rSrcBkt, 1),
				),
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
		CheckDestroy:             testAccCheckWorkflowReplicationDestroy,
		Steps: []resource.TestStep{
			{Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, true, "logs/")},
			{
				ResourceName:                         testAccReplicationResource,
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateBucketAndAttr(testAccReplicationResource, "rule_id")),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "rule_id",
			},
		},
	})
}

// testAccCheckReplicationRules asserts the source bucket's S3 replication
// configuration holds exactly want rules.
func testAccCheckReplicationRules(bucket string, want int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[testAccReplicationResource]
		if !ok {
			return fmt.Errorf("%s not found in state", testAccReplicationResource)
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			return fmt.Errorf("could not get credentials for account %q", rs.Primary.Attributes["account_name"])
		}
		cfg, err := testAccS3Client().GetBucketReplication(context.Background(), acctCreds, bucket)
		if err != nil {
			return err
		}
		got := 0
		if cfg != nil {
			got = len(cfg.Rules)
		}
		if got != want {
			return fmt.Errorf("bucket %s has %d replication rules, want %d", bucket, got, want)
		}
		return nil
	}
}

func testAccCheckWorkflowReplicationDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_bucket_workflow_replication" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		cfg, err := testAccS3Client().GetBucketReplication(context.Background(), acctCreds, rs.Primary.Attributes["bucket_name"])
		// If the bucket itself is gone, GET fails — accept that as destroyed.
		if err != nil || cfg == nil {
			continue
		}
		for _, rule := range cfg.Rules {
			if rule.ID == rs.Primary.Attributes["rule_id"] {
				return fmt.Errorf("replication rule %s on %s still exists", rule.ID, rs.Primary.Attributes["bucket_name"])
			}
		}
	}
	return nil
}

func testAccWorkflowReplicationConfig(acctName, srcLocName, dstLocName, srcBktName, dstBktName string, enabled bool, prefix string) string {
	filter := ""
	if prefix != "" {
		filter = fmt.Sprintf(`

  filter {
    object_key_prefix = %q
  }`, prefix)
	}
	return testAccAccountConfig(acctName) +
		testAccLocationSourceConfig(srcLocName) +
		testAccLocationDestConfig(dstLocName) +
		testAccBucketConfig("source", srcBktName, "artesca_location.source.name", true) +
		testAccBucketConfig("dest", dstBktName, "artesca_location.dest.name", true) +
		fmt.Sprintf(`
resource "artesca_bucket_workflow_replication" "test" {
  account_name            = artesca_account.test.name
  bucket_name             = artesca_bucket.source.name
  destination_bucket_name = artesca_bucket.dest.name
  enabled                 = %t%s
}
`, enabled, filter)
}

func testAccWorkflowReplicationExtraRule(prefix string) string {
	return fmt.Sprintf(`
resource "artesca_bucket_workflow_replication" "second" {
  account_name            = artesca_account.test.name
  bucket_name             = artesca_bucket.source.name
  destination_bucket_name = artesca_bucket.dest.name
  enabled                 = true

  filter {
    object_key_prefix = %q
  }
}
`, prefix)
}
