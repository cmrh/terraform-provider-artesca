package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBucketEncryption_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketEncryptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccBucketConfig("test", rBucket, "null", false) +
					testAccBucketEncryptionConfig("AES256"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_encryption.test", "sse_algorithm", "AES256"),
				),
			},
		},
	})
}

func TestAccBucketEncryption_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketEncryptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccBucketConfig("test", rBucket, "null", false) +
					testAccBucketEncryptionConfig("AES256"),
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

func testAccBucketEncryptionConfig(sseAlgorithm string) string {
	return fmt.Sprintf(`
resource "artesca_bucket_encryption" "test" {
  account_name  = artesca_account.test.name
  bucket_name   = artesca_bucket.test.name
  sse_algorithm = %q
}
`, sseAlgorithm)
}

func testAccCheckBucketEncryptionDestroy(s *terraform.State) error {
	s3Endpoint := os.Getenv("ARTESCA_S3_ENDPOINT")
	if s3Endpoint == "" {
		return nil
	}
	s3Client := testAccS3Client()

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
