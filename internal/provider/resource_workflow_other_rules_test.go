package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
	rLoc := randomName("tf-acc-dloc")
	rSrc := randomName("tf-acc-src")
	rDst := randomName("tf-acc-dst")

	otherRule := fmt.Sprintf(`<ReplicationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`+
		`<Role>%s</Role>`+
		`<Rule><ID>other-loc</ID><Prefix>other/</Prefix><Status>Enabled</Status><Destination><Bucket>arn:aws:s3:::%s</Bucket><StorageClass>%s</StorageClass></Destination></Rule>`+
		`</ReplicationConfiguration>`, client.DefaultReplicationRole, rDst, rLoc)
	kept := fmt.Sprintf(`<StorageClass>%s</StorageClass>`, rLoc)
	base := testAccAccountConfig(rAcct) + testAccLocationDestConfig(rLoc) +
		testAccBucketConfig("source", rSrc, "artesca_location.dest.name", true) +
		testAccBucketConfig("dest", rDst, "artesca_location.dest.name", true)
	withRule := func(enabled bool) string {
		return base + fmt.Sprintf(`
resource "artesca_bucket_workflow_replication" "test" {
  account_name            = artesca_account.test.name
  bucket_name             = artesca_bucket.source.name
  destination_bucket_name = artesca_bucket.dest.name
  enabled                 = %t

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
		return c.PutBucketSubresourceXML(context.Background(), creds, bucket, subresource, body)
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
