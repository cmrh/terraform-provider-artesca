package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBucket_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket.test", "name", rBucket),
					resource.TestCheckResourceAttr("artesca_bucket.test", "location_constraint", rLoc),
					resource.TestCheckResourceAttr("artesca_bucket.test", "versioning_enabled", "false"),
				),
			},
		},
	})
}

func TestAccBucket_versioning(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket.test", "versioning_enabled", "false"),
				),
			},
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket.test", "versioning_enabled", "true"),
				),
			},
		},
	})
}
