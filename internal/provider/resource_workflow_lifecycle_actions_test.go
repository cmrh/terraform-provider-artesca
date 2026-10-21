package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Noncurrent-version, multipart-upload, delete-marker and date expirations,
// with tag filters, on ARTESCA's own storage.
func TestAccWorkflowExpiration_actions(t *testing.T) {
	rAcct, rBucket := randomName("tf-acc"), randomName("tf-acc-bkt")
	base := testAccAccountConfig(rAcct) + testAccBucketConfig("test", rBucket, "null", true)
	expiration := func(body string) string {
		return base + fmt.Sprintf(`
resource "artesca_bucket_workflow_expiration" "test" {
  account_name = artesca_account.test.name
  bucket_name  = artesca_bucket.test.name
  enabled      = true
%s
}
`, body)
	}
	const r = "artesca_bucket_workflow_expiration.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: expiration(`
  current_version_trigger_delay_days             = 30
  previous_version_trigger_delay_days            = 10
  incomplete_multipart_upload_trigger_delay_days = 7
  filter {
    object_key_prefix = "logs/"
  }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(r, "previous_version_trigger_delay_days", "10"),
					resource.TestCheckResourceAttr(r, "incomplete_multipart_upload_trigger_delay_days", "7"),
					testAccCheckBucketXMLContains("artesca_bucket.test", "lifecycle",
						"<NoncurrentDays>10</NoncurrentDays>", "<DaysAfterInitiation>7</DaysAfterInitiation>", "<Days>30</Days>"),
				),
			},
			{
				// Switch to a date with a tag filter; the multipart action goes.
				Config: expiration(`
  current_version_trigger_delay_date  = "2030-01-01"
  previous_version_trigger_delay_days = 10
  filter {
    object_key_prefix = "logs/"
    object_tags       = { team = "data", tier = "cold" }
  }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(r, "current_version_trigger_delay_date", "2030-01-01"),
					resource.TestCheckNoResourceAttr(r, "current_version_trigger_delay_days"),
					resource.TestCheckNoResourceAttr(r, "incomplete_multipart_upload_trigger_delay_days"),
					resource.TestCheckResourceAttr(r, "filter.object_tags.team", "data"),
					testAccCheckBucketXMLContains("artesca_bucket.test", "lifecycle",
						"<Date>2030-01-01T00:00:00", "<Key>team</Key>", "<Key>tier</Key>", "<Prefix>logs/</Prefix>"),
				),
			},
			{
				ResourceName:                         r,
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateBucketAndAttr(r, "rule_id")),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "rule_id",
			},
			{
				Config: expiration(`
  expire_delete_markers = true`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(r, "expire_delete_markers", "true"),
					resource.TestCheckNoResourceAttr(r, "previous_version_trigger_delay_days"),
					testAccCheckBucketXMLContains("artesca_bucket.test", "lifecycle",
						"<ExpiredObjectDeleteMarker>true</ExpiredObjectDeleteMarker>"),
				),
			},
		},
	})
}

// Noncurrent-version and date transitions with a tag filter.
func TestAccWorkflowTransition_noncurrentAndDate(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rBucket := randomName("tf-acc-bkt")
	base := testAccAccountConfig(rAcct) +
		testAccLocationSourceConfig(rSrcLoc) +
		testAccLocationDestConfig(rDstLoc) +
		testAccBucketConfig("test", rBucket, "artesca_location.source.name", true)
	transition := func(body string) string {
		return base + fmt.Sprintf(`
resource "artesca_bucket_workflow_transition" "test" {
  account_name  = artesca_account.test.name
  bucket_name   = artesca_bucket.test.name
  enabled       = true
  location_name = artesca_location.dest.name
%s
}
`, body)
	}
	const r = "artesca_bucket_workflow_transition.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: transition(`
  trigger_delay_date = "2030-01-01"
  filter {
    object_tags = { team = "data" }
  }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(r, "apply_to_version", "current"),
					resource.TestCheckResourceAttr(r, "trigger_delay_date", "2030-01-01"),
					resource.TestCheckNoResourceAttr(r, "trigger_delay_days"),
					testAccCheckBucketXMLContains("artesca_bucket.test", "lifecycle",
						"<Transition><Date>2030-01-01T00:00:00", "<Filter><Tag><Key>team</Key>"),
				),
			},
			{
				// Changing apply_to_version replaces the rule.
				Config: transition(`
  apply_to_version   = "noncurrent"
  trigger_delay_days = 0`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(r, "apply_to_version", "noncurrent"),
					resource.TestCheckResourceAttr(r, "trigger_delay_days", "0"),
					testAccCheckBucketXMLContains("artesca_bucket.test", "lifecycle",
						"<NoncurrentVersionTransition><NoncurrentDays>0</NoncurrentDays>"),
				),
			},
			{
				ResourceName:                         r,
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateBucketAndAttr(r, "rule_id")),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "rule_id",
			},
		},
	})
}
