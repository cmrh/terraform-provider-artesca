package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Rules created outside Terraform with elements the provider doesn't model
// must survive the workflow resources creating, updating, and deleting their
// own rules on the same bucket.

func TestAccWorkflowExpiration_keepsOtherRules(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-dloc")
	rBucket := randomName("tf-acc-bkt")

	otherRules := `<LifecycleConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
		`<Rule><ID>other-tag</ID><Status>Enabled</Status><Filter><Tag><Key>env</Key><Value>dev</Value></Tag></Filter><Expiration><Days>7</Days></Expiration></Rule>` +
		`<Rule><ID>other-noncurrent</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter><NoncurrentVersionExpiration><NoncurrentDays>10</NoncurrentDays></NoncurrentVersionExpiration></Rule>` +
		`</LifecycleConfiguration>`
	keptFragments := []string{
		`<Tag><Key>env</Key><Value>dev</Value></Tag>`,
		`<NoncurrentVersionExpiration><NoncurrentDays>10</NoncurrentDays></NoncurrentVersionExpiration>`,
	}
	base := testAccAccountConfig(rAcct) + testAccLocationDestConfig(rLoc) +
		testAccBucketConfig("test", rBucket, "artesca_location.dest.name", false)
	withRule := func(days int) string {
		return base + fmt.Sprintf(`
resource "artesca_bucket_workflow_expiration" "test" {
  account_name                       = artesca_account.test.name
  bucket_name                        = artesca_bucket.test.name
  enabled                            = true
  current_version_trigger_delay_days = %d

  filter {
    object_key_prefix = "tf/"
  }
}
`, days)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: base,
				Check:  testAccPutBucketXML("artesca_bucket.test", "lifecycle", otherRules),
			},
			{
				Config: withRule(30),
				Check:  testAccCheckBucketXMLContains("artesca_bucket.test", "lifecycle", append(keptFragments, "<Days>30</Days>")...),
			},
			{
				Config: withRule(45),
				Check:  testAccCheckBucketXMLContains("artesca_bucket.test", "lifecycle", append(keptFragments, "<Days>45</Days>")...),
			},
			{
				Config: base,
				Check:  testAccCheckBucketXMLContains("artesca_bucket.test", "lifecycle", keptFragments...),
			},
		},
	})
}

func TestAccWorkflowReplication_keepsOtherRules(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rSrc := randomName("tf-acc-src")

	// Both rules replicate the us-east-1 source bucket to the RING location,
	// under different prefixes. The out-of-band rule names its location in
	// StorageClass, which the provider must write back unchanged.
	otherRule := fmt.Sprintf(`<ReplicationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`+
		`<Role>%s</Role>`+
		`<Rule><ID>other-loc</ID><Prefix>other/</Prefix><Status>Enabled</Status><Destination><Bucket>arn:aws:s3:::%s</Bucket><StorageClass>%s</StorageClass></Destination></Rule>`+
		`</ReplicationConfiguration>`, client.DefaultReplicationRole, rSrc, rLoc)
	kept := fmt.Sprintf(`<ID>other-loc</ID><Prefix>other/</Prefix><Status>Enabled</Status><Destination><Bucket>arn:aws:s3:::%s</Bucket><StorageClass>%s</StorageClass></Destination>`, rSrc, rLoc)
	base := testAccAccountConfig(rAcct) + testAccLocationSourceConfig(rLoc) +
		testAccBucketConfig("source", rSrc, "null", true)
	withRule := func(enabled bool) string {
		return base + fmt.Sprintf(`
resource "artesca_bucket_workflow_replication" "test" {
  account_name         = artesca_account.test.name
  bucket_name          = artesca_bucket.source.name
  destination_location = artesca_location.source.name
  enabled              = %t

  filter {
    object_key_prefix = "tf/"
  }
}
`, enabled)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: base,
				Check:  testAccPutBucketXML("artesca_bucket.source", "replication", otherRule),
			},
			{
				Config: withRule(true),
				Check:  testAccCheckBucketXMLContains("artesca_bucket.source", "replication", kept, "<Prefix>tf/</Prefix><Status>Enabled</Status>"),
			},
			{
				Config: withRule(false),
				Check:  testAccCheckBucketXMLContains("artesca_bucket.source", "replication", kept, "<Prefix>tf/</Prefix><Status>Disabled</Status>"),
			},
			{
				Config: base,
				Check:  testAccCheckBucketXMLContains("artesca_bucket.source", "replication", kept),
			},
		},
	})
}

func testAccBucketS3(s *terraform.State, bucketResource string) (*client.S3Client, client.Credentials, string, error) {
	rs, ok := s.RootModule().Resources[bucketResource]
	if !ok {
		return nil, client.Credentials{}, "", fmt.Errorf("%s not in state", bucketResource)
	}
	creds, ok := testAccAccountCredentials(rs)
	if !ok {
		return nil, client.Credentials{}, "", fmt.Errorf("no credentials for %s", rs.Primary.Attributes["account_name"])
	}
	return testAccS3Client(), creds, rs.Primary.Attributes["name"], nil
}

func testAccPutBucketXML(bucketResource, subresource, body string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c, creds, bucket, err := testAccBucketS3(s, bucketResource)
		if err != nil {
			return err
		}
		// A rule naming a location created moments earlier is rejected with
		// MalformedXML until the location propagates.
		deadline := time.Now().Add(3 * time.Minute)
		for {
			err := c.PutBucketSubresourceXML(context.Background(), creds, bucket, subresource, body)
			if err == nil || !strings.Contains(err.Error(), "MalformedXML") || time.Now().After(deadline) {
				return err
			}
			time.Sleep(10 * time.Second)
		}
	}
}

func testAccCheckBucketXMLContains(bucketResource, subresource string, fragments ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c, creds, bucket, err := testAccBucketS3(s, bucketResource)
		if err != nil {
			return err
		}
		got, err := c.GetBucketSubresourceXML(context.Background(), creds, bucket, subresource)
		if err != nil {
			return err
		}
		for _, f := range fragments {
			if !strings.Contains(got, f) {
				return fmt.Errorf("bucket %s %s missing %s:\n%s", bucket, subresource, f, got)
			}
		}
		return nil
	}
}

// A rule created outside Terraform keeps the parts the resource doesn't model
// (a tag in an And filter, a noncurrent-version action) when imported and
// updated.
func TestAccWorkflowExpiration_importKeepsUnmodelledParts(t *testing.T) {
	rAcct, rBucket := randomName("tf-acc"), randomName("tf-acc-bkt")
	base := testAccAccountConfig(rAcct) + testAccBucketConfig("test", rBucket, "null", true)
	rule := `<LifecycleConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Rule><ID>outside</ID>` +
		`<Filter><And><Prefix>logs/</Prefix><Tag><Key>team</Key><Value>a</Value></Tag></And></Filter><Status>Enabled</Status>` +
		`<Expiration><Days>30</Days></Expiration><NoncurrentVersionExpiration><NoncurrentDays>7</NoncurrentDays></NoncurrentVersionExpiration>` +
		`</Rule></LifecycleConfiguration>`
	managed := func(days int) string {
		return base + fmt.Sprintf(`
import {
  to = artesca_bucket_workflow_expiration.test
  id = "%s/%s/outside"
}

resource "artesca_bucket_workflow_expiration" "test" {
  account_name = artesca_account.test.name
  bucket_name  = artesca_bucket.test.name
  rule_id      = "outside"
  enabled      = true

  current_version_trigger_delay_days = %d

  filter {
    object_key_prefix = "logs/"
  }
}
`, rAcct, rBucket, days)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{Config: base, Check: testAccPutBucketXML("artesca_bucket.test", "lifecycle", rule)},
			{
				// The prefix inside Filter/And is read, so the import plans nothing.
				Config: managed(30),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("artesca_bucket_workflow_expiration.test", plancheck.ResourceActionNoop)},
				},
			},
			{
				Config: managed(45),
				Check: testAccCheckBucketXMLContains("artesca_bucket.test", "lifecycle",
					"<Days>45</Days>", "<Key>team</Key>", "<Prefix>logs/</Prefix>", "<NoncurrentDays>7</NoncurrentDays>"),
			},
		},
	})
}
