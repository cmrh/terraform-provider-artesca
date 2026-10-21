package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceBucket_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) + fmt.Sprintf(`
resource "artesca_bucket" "test" {
  account_name        = artesca_account.test.name
  name                = %q
  object_lock_enabled = true
}

resource "artesca_bucket_tagging" "test" {
  account_name = artesca_account.test.name
  bucket_name  = artesca_bucket.test.name
  tags         = { team = "data" }
}

data "artesca_bucket" "test" {
  account_name = artesca_account.test.name
  name         = artesca_bucket_tagging.test.bucket_name
}
`, rBucket),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artesca_bucket.test", "arn", "arn:aws:s3:::"+rBucket),
					resource.TestCheckResourceAttr("data.artesca_bucket.test", "location_constraint", "us-east-1"),
					resource.TestCheckResourceAttr("data.artesca_bucket.test", "versioning_enabled", "true"),
					resource.TestCheckResourceAttr("data.artesca_bucket.test", "object_lock_enabled", "true"),
					resource.TestCheckResourceAttr("data.artesca_bucket.test", "tags.team", "data"),
				),
			},
		},
	})
}

func TestAccDataSourceBucket_notFound(t *testing.T) {
	rAcct := randomName("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) + `
data "artesca_bucket" "test" {
  account_name = artesca_account.test.name
  name         = "tf-acc-no-such-bucket"
}
`,
				ExpectError: regexp.MustCompile(`Bucket not found`),
			},
		},
	})
}
