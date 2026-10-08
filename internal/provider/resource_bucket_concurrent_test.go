package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Several bucket-config resources (tagging, encryption, policy, lifecycle) on
// the same bucket are created in parallel.
// ARTESCA can drop concurrent config writes to one bucket, so the provider
// serializes them per bucket; every config must survive the apply and the
// post-apply plan must be empty.
func TestAccBucket_concurrentConfigs(t *testing.T) {
	rAcct := randomName("tf-acc")
	rPrefix := randomName("tf-acc-conc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) + testAccBucketConcurrentConfig(rPrefix),
				Check:  testAccCheckBucketConfigsPresent(rPrefix, 3),
			},
		},
	})
}

func testAccBucketConcurrentConfig(prefix string) string {
	return fmt.Sprintf(`
resource "artesca_bucket" "src" {
  count              = 3
  account_name       = artesca_account.test.name
  name               = "%[1]s-${count.index}"
  versioning_enabled = true
}

resource "artesca_bucket_tagging" "src" {
  count        = 3
  account_name = artesca_account.test.name
  bucket_name  = artesca_bucket.src[count.index].name
  tags         = { env = "concurrency" }
}

resource "artesca_bucket_encryption" "src" {
  count              = 3
  account_name       = artesca_account.test.name
  bucket_name        = artesca_bucket.src[count.index].name
  sse_algorithm      = "AES256"
  bucket_key_enabled = false
}

resource "artesca_bucket_policy" "src" {
  count        = 3
  account_name = artesca_account.test.name
  bucket_name  = artesca_bucket.src[count.index].name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = "arn:aws:iam::${artesca_account.test.id}:root" }
      Action    = ["s3:GetObject"]
      Resource  = "arn:aws:s3:::${artesca_bucket.src[count.index].name}/*"
    }]
  })
}

resource "artesca_bucket_workflow_expiration" "src" {
  count                              = 3
  account_name                       = artesca_account.test.name
  bucket_name                        = artesca_bucket.src[count.index].name
  enabled                            = true
  current_version_trigger_delay_days = 90
  filter {
    object_key_prefix = "logs/"
  }
}
`, prefix)
}

// testAccCheckBucketConfigsPresent reads every config back through S3.
func testAccCheckBucketConfigsPresent(prefix string, n int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["artesca_bucket.src.0"]
		if !ok {
			return fmt.Errorf("artesca_bucket.src.0 not found in state")
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			return fmt.Errorf("could not get account credentials")
		}
		c := testAccS3Client()
		ctx := context.Background()
		var missing []string
		for i := 0; i < n; i++ {
			bucket := fmt.Sprintf("%s-%d", prefix, i)
			if tags, err := c.GetBucketTagging(ctx, acctCreds, bucket); err != nil || len(tags) == 0 {
				missing = append(missing, bucket+":tagging")
			}
			if enc, err := c.GetBucketEncryption(ctx, acctCreds, bucket); err != nil || enc == nil {
				missing = append(missing, bucket+":encryption")
			}
			if pol, err := c.GetBucketPolicy(ctx, acctCreds, bucket); err != nil || pol == "" {
				missing = append(missing, bucket+":policy")
			}
			if rules, err := c.GetBucketLifecycle(ctx, acctCreds, bucket); err != nil || len(rules) != 1 {
				missing = append(missing, bucket+":lifecycle")
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("bucket configs missing after apply: %v", missing)
		}
		return nil
	}
}
