package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWorkflowTransition_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowTransitionConfig(rAcct, rSrcLoc, rDstLoc, rBucket, 60, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_workflow_transition.test", "bucket_name", rBucket),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_transition.test", "enabled", "true"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_transition.test", "trigger_delay_days", "60"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_transition.test", "location_name", rDstLoc),
					resource.TestCheckResourceAttrSet("artesca_bucket_workflow_transition.test", "rule_id"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_transition.test", "filter.object_key_prefix", "archive/"),
				),
			},
		},
	})
}

func TestAccWorkflowTransition_update(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowTransitionConfig(rAcct, rSrcLoc, rDstLoc, rBucket, 60, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_workflow_transition.test", "trigger_delay_days", "60"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_transition.test", "enabled", "true"),
				),
			},
			{
				Config: testAccWorkflowTransitionConfig(rAcct, rSrcLoc, rDstLoc, rBucket, 90, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_workflow_transition.test", "trigger_delay_days", "90"),
					resource.TestCheckResourceAttr("artesca_bucket_workflow_transition.test", "enabled", "false"),
				),
			},
		},
	})
}

func testAccWorkflowTransitionConfig(acctName, srcLocName, dstLocName, bucketName string, days int, enabled bool) string {
	return testAccAccountConfig(acctName) +
		testAccLocationSourceConfig(srcLocName) +
		testAccLocationDestConfig(dstLocName) +
		testAccBucketConfig("test", bucketName, "artesca_location.source.name", false) +
		fmt.Sprintf(`
resource "artesca_bucket_workflow_transition" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  bucket_name        = artesca_bucket.test.name
  enabled            = %t
  location_name      = artesca_location.dest.name
  trigger_delay_days = %d

  filter {
    object_key_prefix = "archive/"
  }
}
`, enabled, days)
}
