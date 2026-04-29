package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWorkflowExpiration_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowExpirationConfig(rAcct, rLoc, rBucket, 30, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_workflow_expiration.test", "bucket_name", rBucket),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_expiration.test", "enabled", "true"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_expiration.test", "current_version_trigger_delay_days", "30"),
					resource.TestCheckResourceAttrSet("artesca_bucket_workflow_expiration.test", "rule_id"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_expiration.test", "filter.object_key_prefix", "logs/"),
				),
			},
		},
	})
}

func TestAccWorkflowExpiration_update(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowExpirationConfig(rAcct, rLoc, rBucket, 30, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_workflow_expiration.test", "current_version_trigger_delay_days", "30"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_expiration.test", "enabled", "true"),
				),
			},
			{
				Config: testAccWorkflowExpirationConfig(rAcct, rLoc, rBucket, 60, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_workflow_expiration.test", "current_version_trigger_delay_days", "60"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_expiration.test", "enabled", "false"),
				),
			},
		},
	})
}

func testAccWorkflowExpirationConfig(acctName, locName, bucketName string, days int, enabled bool) string {
	return testAccAccountConfig(acctName) +
		testAccLocationSourceConfig(locName) +
		testAccBucketConfig("test", bucketName, "artesca_location.source.name", false) +
		fmt.Sprintf(`
resource "artesca_bucket_workflow_expiration" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  bucket_name        = artesca_bucket.test.name
  enabled            = %t

  current_version_trigger_delay_days = %d

  filter {
    object_key_prefix = "logs/"
  }
}
`, enabled, days)
}
