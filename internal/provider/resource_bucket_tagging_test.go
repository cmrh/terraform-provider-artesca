package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBucketTagging_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketTaggingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketTaggingConfig(map[string]string{"environment": "prod", "team": "data"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_tagging.test", "tags.environment", "prod"),
					resource.TestCheckResourceAttr("artesca_bucket_tagging.test", "tags.team", "data"),
				),
			},
		},
	})
}

func TestAccBucketTagging_update(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketTaggingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketTaggingConfig(map[string]string{"environment": "prod"}),
			},
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketTaggingConfig(map[string]string{"environment": "staging", "owner": "alice"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_tagging.test", "tags.environment", "staging"),
					resource.TestCheckResourceAttr("artesca_bucket_tagging.test", "tags.owner", "alice"),
					resource.TestCheckNoResourceAttr("artesca_bucket_tagging.test", "tags.team"),
				),
			},
		},
	})
}

func TestAccBucketTagging_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketTaggingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false) +
					testAccBucketTaggingConfig(map[string]string{"environment": "prod"}),
			},
			{
				ResourceName:                         "artesca_bucket_tagging.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateID(rBucket)),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket_name",
			},
		},
	})
}

func testAccBucketTaggingConfig(tags map[string]string) string {
	body := "{\n"
	for k, v := range tags {
		body += fmt.Sprintf("    %q = %q\n", k, v)
	}
	body += "  }"
	return fmt.Sprintf(`
resource "artesca_bucket_tagging" "test" {
  account_name = artesca_account.test.name
  bucket_name        = artesca_bucket.test.name
  tags = %s
}
`, body)
}

func testAccCheckBucketTaggingDestroy(s *terraform.State) error {
	s3Endpoint := os.Getenv("ARTESCA_S3_ENDPOINT")
	if s3Endpoint == "" {
		return nil
	}
	s3Client := testAccS3Client()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_bucket_tagging" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		tags, err := s3Client.GetBucketTagging(context.Background(),
			acctCreds,
			rs.Primary.Attributes["bucket_name"])
		// If the bucket itself is gone, GET will fail — accept that as destroyed.
		if err != nil {
			continue
		}
		if len(tags) > 0 {
			return fmt.Errorf("bucket tags on %s still exist: %v", rs.Primary.Attributes["bucket_name"], tags)
		}
	}
	return nil
}
