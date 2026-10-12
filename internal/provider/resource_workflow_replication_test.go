package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testAccReplicationResource = "artesca_bucket_workflow_replication.test"

// Replication rules send a bucket on us-east-1 to a remote location, the main
// use case. The location is created in the same apply, which also exercises the
// retry while it propagates.
func TestAccWorkflowReplication_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rSrcBkt := randomName("tf-acc-sbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowReplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rLoc, rSrcBkt, true, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccReplicationResource, "enabled", "true"),
					resource.TestCheckResourceAttr(testAccReplicationResource, "bucket_name", rSrcBkt),
					resource.TestCheckResourceAttr(testAccReplicationResource, "destination_location", rLoc),
					resource.TestCheckNoResourceAttr(testAccReplicationResource, "destination_bucket_name"),
					resource.TestCheckResourceAttrSet(testAccReplicationResource, "rule_id"),
					testAccCheckReplicationRules(rSrcBkt, 1),
					testAccCheckBucketXMLContains("artesca_bucket.source", "replication",
						fmt.Sprintf("<Bucket>arn:aws:s3:::%s</Bucket><StorageClass>%s</StorageClass>", rSrcBkt, rLoc)),
				),
			},
		},
	})
}

func TestAccWorkflowReplication_update(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rSrcBkt := randomName("tf-acc-sbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowReplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rLoc, rSrcBkt, true, ""),
				Check:  resource.TestCheckResourceAttr(testAccReplicationResource, "enabled", "true"),
			},
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rLoc, rSrcBkt, false, "logs/"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccReplicationResource, "enabled", "false"),
					resource.TestCheckResourceAttr(testAccReplicationResource, "filter.object_key_prefix", "logs/"),
				),
			},
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rLoc, rSrcBkt, true, ""),
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
	rLoc := randomName("tf-acc-loc")
	rSrcBkt := randomName("tf-acc-sbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowReplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rLoc, rSrcBkt, true, "a/") +
					testAccWorkflowReplicationExtraRule("b/"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccReplicationResource, "filter.object_key_prefix", "a/"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.second", "filter.object_key_prefix", "b/"),
					testAccCheckReplicationRules(rSrcBkt, 2),
				),
			},
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rLoc, rSrcBkt, true, "a/"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccReplicationResource, "filter.object_key_prefix", "a/"),
					testAccCheckReplicationRules(rSrcBkt, 1),
				),
			},
		},
	})
}

// A bucket on one remote location (lab1) replicates to another (lab2).
func TestAccWorkflowReplication_remoteToRemote(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rSrcBkt := randomName("tf-acc-sbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowReplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rSrcLoc) +
					testAccLocationDestConfig(rDstLoc) +
					testAccBucketConfig("source", rSrcBkt, "artesca_location.source.name", true) + `
resource "artesca_bucket_workflow_replication" "test" {
  account_name         = artesca_account.test.name
  bucket_name          = artesca_bucket.source.name
  destination_location = artesca_location.dest.name
  enabled              = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccReplicationResource, "destination_location", rDstLoc),
					resource.TestCheckNoResourceAttr(testAccReplicationResource, "destination_bucket_name"),
					testAccCheckBucketXMLContains("artesca_bucket.source", "replication",
						fmt.Sprintf("<Bucket>arn:aws:s3:::%s</Bucket><StorageClass>%s</StorageClass>", rSrcBkt, rDstLoc)),
				),
			},
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

func TestAccWorkflowReplication_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rSrcBkt := randomName("tf-acc-sbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowReplicationDestroy,
		Steps: []resource.TestStep{
			{Config: testAccWorkflowReplicationConfig(rAcct, rLoc, rSrcBkt, true, "logs/")},
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

// testAccWorkflowReplicationConfig replicates a bucket on us-east-1, ARTESCA's
// own storage, to a RING location.
func testAccWorkflowReplicationConfig(acctName, locName, srcBktName string, enabled bool, prefix string) string {
	filter := ""
	if prefix != "" {
		filter = fmt.Sprintf(`

  filter {
    object_key_prefix = %q
  }`, prefix)
	}
	return testAccAccountConfig(acctName) +
		testAccLocationSourceConfig(locName) +
		testAccBucketConfig("source", srcBktName, "null", true) +
		fmt.Sprintf(`
resource "artesca_bucket_workflow_replication" "test" {
  account_name         = artesca_account.test.name
  bucket_name          = artesca_bucket.source.name
  destination_location = artesca_location.source.name
  enabled              = %t%s
}
`, enabled, filter)
}

func testAccWorkflowReplicationExtraRule(prefix string) string {
	return fmt.Sprintf(`
resource "artesca_bucket_workflow_replication" "second" {
  account_name         = artesca_account.test.name
  bucket_name          = artesca_bucket.source.name
  destination_location = artesca_location.source.name
  enabled              = true

  filter {
    object_key_prefix = %q
  }
}
`, prefix)
}
