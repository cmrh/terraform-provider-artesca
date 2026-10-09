package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBucketPolicy_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketPolicyConfig(rBucket, "s3:GetObject"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_policy.test", "bucket_name", rBucket),
					resource.TestCheckResourceAttrSet("artesca_bucket_policy.test", "policy"),
				),
			},
		},
	})
}

func TestAccBucketPolicy_update(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketPolicyConfig(rBucket, "s3:GetObject"),
			},
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketPolicyConfig(rBucket, "s3:GetObject,s3:PutObject"),
			},
		},
	})
}

// TestAccBucketPolicy_jsonEquivalence verifies that re-applying a policy
// with whitespace and key-order differences is a no-op (planEquivalence
// modifier suppresses the spurious diff).
func TestAccBucketPolicy_jsonEquivalence(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	compactPolicy := `{"Version":"2012-10-17","Statement":[{"Sid":"S","Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::` + rBucket + `/*"}]}`
	prettyPolicy := `{
  "Statement": [
    {
      "Action": "s3:GetObject",
      "Effect": "Allow",
      "Principal": "*",
      "Resource": "arn:aws:s3:::` + rBucket + `/*",
      "Sid": "S"
    }
  ],
  "Version": "2012-10-17"
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketPolicyRawConfig(rBucket, compactPolicy),
			},
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketPolicyRawConfig(rBucket, prettyPolicy),
				PlanOnly: true,
			},
		},
	})
}

func TestAccBucketPolicy_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketPolicyConfig(rBucket, "s3:GetObject"),
			},
			{
				ResourceName:                         "artesca_bucket_policy.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateID(rBucket)),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket_name",
				ImportStateVerifyIgnore:              []string{"policy"},
			},
		},
	})
}

func testAccBucketPolicyConfig(bucketName, action string) string {
	return fmt.Sprintf(`
resource "artesca_bucket_policy" "test" {
  account_name = artesca_account.test.name
  bucket_name        = artesca_bucket.test.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "S"
      Effect    = "Allow"
      Principal = "*"
      Action    = split(",", %q)
      Resource  = "arn:aws:s3:::%s/*"
    }]
  })
}
`, action, bucketName)
}

func testAccBucketPolicyRawConfig(bucketName, rawPolicy string) string {
	_ = bucketName
	// Use heredoc so embedded JSON braces don't conflict with HCL interpolation.
	return fmt.Sprintf(`
resource "artesca_bucket_policy" "test" {
  account_name = artesca_account.test.name
  bucket_name        = artesca_bucket.test.name

  policy = <<POLICY
%s
POLICY
}
`, rawPolicy)
}

func testAccCheckBucketPolicyDestroy(s *terraform.State) error {
	s3Endpoint := os.Getenv("ARTESCA_S3_ENDPOINT")
	if s3Endpoint == "" {
		return nil
	}
	s3Client := testAccS3Client()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_bucket_policy" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		policy, err := s3Client.GetBucketPolicy(context.Background(),
			acctCreds,
			rs.Primary.Attributes["bucket_name"])
		// If the bucket itself is gone, GET will fail — accept that as destroyed.
		if err != nil {
			continue
		}
		if policy != "" {
			return fmt.Errorf("bucket policy on %s still exists", rs.Primary.Attributes["bucket_name"])
		}
	}
	return nil
}
