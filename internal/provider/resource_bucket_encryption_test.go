package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
)

func TestAccBucketEncryption_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketEncryptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketEncryptionConfig("AES256", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_encryption.test", "sse_algorithm", "AES256"),
					resource.TestCheckResourceAttr("artesca_bucket_encryption.test", "bucket_key_enabled", "false"),
				),
			},
		},
	})
}

func TestAccBucketEncryption_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketEncryptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketEncryptionConfig("AES256", false),
			},
			{
				ResourceName:                         "artesca_bucket_encryption.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateID(rBucket)),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket_name",
			},
		},
	})
}

func testAccBucketEncryptionConfig(sseAlgorithm string, bucketKeyEnabled bool) string {
	return fmt.Sprintf(`
resource "artesca_bucket_encryption" "test" {
  account_name = artesca_account.test.name
  bucket_name         = artesca_bucket.test.name
  sse_algorithm       = %q
  bucket_key_enabled  = %t
}
`, sseAlgorithm, bucketKeyEnabled)
}

func testAccCheckBucketEncryptionDestroy(s *terraform.State) error {
	s3Endpoint := os.Getenv("ARTESCA_S3_ENDPOINT")
	if s3Endpoint == "" {
		return nil
	}
	insecure := os.Getenv("ARTESCA_INSECURE_SKIP_VERIFY") == "true" || os.Getenv("ARTESCA_INSECURE_SKIP_VERIFY") == "1"
	s3Client := client.NewS3Client(s3Endpoint, "us-east-1", insecure)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_bucket_encryption" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		cfg, err := s3Client.GetBucketEncryption(context.Background(),
			acctCreds,
			rs.Primary.Attributes["bucket_name"])
		// If the bucket itself is gone, GET will fail — accept that as destroyed.
		if err != nil {
			continue
		}
		if cfg != nil {
			return fmt.Errorf("bucket encryption on %s still present: %+v", rs.Primary.Attributes["bucket_name"], cfg)
		}
	}
	return nil
}
