package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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

// testAccCRRLocationsConfig adds a RING S3 location on lab2 and a CRR
// location on lab1 next to an account and a versioned bucket.
func testAccCRRLocationsConfig(acct, bucket, ringLoc, crrLoc string) string {
	return testAccAccountConfig(acct) + testAccBucketConfig("source", bucket, "null", true) + fmt.Sprintf(`
resource "artesca_location" "ring" {
  name          = %q
  location_type = "location-scality-ring-s3-v1"
  details {
    endpoint     = %q
    access_key   = %q
    secret_key   = %q
    bucket_name  = %q
    bucket_match = false
  }
}

resource "artesca_location" "crr" {
  name          = %q
  location_type = "location-scality-crr-v1"
  details {
    endpoint     = %q
    sts_endpoint = %q
    access_key   = %q
    secret_key   = %q
  }
}
`, ringLoc, os.Getenv("TF_VAR_dest_ring_s3_endpoint"), os.Getenv("TF_VAR_dest_ring_s3_access_key"), os.Getenv("TF_VAR_dest_ring_s3_secret_key"), os.Getenv("TF_VAR_dest_ring_s3_bucket_name"),
		crrLoc, os.Getenv("TF_VAR_ring_s3_endpoint"), os.Getenv("TF_VAR_ring_s3_endpoint"), os.Getenv("TF_VAR_ring_s3_access_key"), os.Getenv("TF_VAR_ring_s3_secret_key"))
}

func testAccCRRRule(name, location, role string) string {
	roleLine := ""
	if role != "" {
		roleLine = fmt.Sprintf("\n  destination_role        = %q", role)
	}
	return fmt.Sprintf(`
resource "artesca_bucket_workflow_replication" %q {
  account_name            = artesca_account.test.name
  bucket_name             = artesca_bucket.source.name
  destination_location    = %s
  destination_bucket_name = "tf-acc-crr-dest"
  enabled                 = true%s

  filter {
    object_key_prefix = "%s/"
  }
}
`, name, location, roleLine, name)
}

func TestAccWorkflowReplication_crrDestinationRole(t *testing.T) {
	rAcct, rBkt, rRing, rCRR := randomName("tf-acc"), randomName("tf-acc-sbkt"), randomName("tf-acc-ring"), randomName("tf-acc-crr")
	base := testAccCRRLocationsConfig(rAcct, rBkt, rRing, rCRR)
	src := client.DefaultReplicationRole
	role1, role2 := "arn:aws:iam::111111111111:role/crr-role", "arn:aws:iam::111111111111:role/crr-role-2"
	// ARTESCA requires every rule on a bucket to name the same destination
	// bucket; a RING location ignores it and writes to its own target bucket.
	ringRule := testAccCRRRule("ring", "artesca_location.ring.name", "")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t); testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkflowReplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: base + testAccCRRRule("crr", "artesca_location.crr.name", role1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_workflow_replication.crr", "destination_role", role1),
					testAccCheckBucketXMLContains("artesca_bucket.source", "replication", "<Role>"+src+","+role1+"</Role>"),
				),
			},
			{
				// A RING rule next to the CRR rule shares the two-ARN role.
				Config: base + testAccCRRRule("crr", "artesca_location.crr.name", role1) + ringRule,
				Check:  testAccCheckBucketXMLContains("artesca_bucket.source", "replication", "<Role>"+src+","+role1+"</Role>"),
			},
			{
				Config: base + testAccCRRRule("crr", "artesca_location.crr.name", role2) + ringRule,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("artesca_bucket_workflow_replication.crr", plancheck.ResourceActionUpdate)},
				},
				Check: testAccCheckBucketXMLContains("artesca_bucket.source", "replication", "<Role>"+src+","+role2+"</Role>"),
			},
			{
				ResourceName:                         "artesca_bucket_workflow_replication.crr",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateBucketAndAttr("artesca_bucket_workflow_replication.crr", "rule_id")),
				ImportStateVerifyIdentifierAttribute: "rule_id",
			},
			{
				// Removing the last CRR rule drops the destination role.
				Config: base + ringRule,
				Check:  testAccCheckBucketXMLContains("artesca_bucket.source", "replication", "<Role>"+src+"</Role>"),
			},
			{
				Config:      base + ringRule + testAccCRRRule("crr", "artesca_location.crr.name", ""),
				ExpectError: regexp.MustCompile(`Missing destination_role`),
			},
			{
				Config:      base + ringRule + testAccCRRRule("ring2", "artesca_location.ring.name", role1),
				ExpectError: regexp.MustCompile(`Unexpected destination_role`),
			},
		},
	})
}
