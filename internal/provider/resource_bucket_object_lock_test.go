package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccBucketObjectLock_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) + testAccBucketObjectLockConfig(rBucket, "GOVERNANCE", "retention_days = 1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket.test", "object_lock_enabled", "true"),
					resource.TestCheckResourceAttr("artesca_bucket.test", "versioning_enabled", "true"),
					resource.TestCheckResourceAttr("artesca_bucket_object_lock.test", "retention_mode", "GOVERNANCE"),
					resource.TestCheckResourceAttr("artesca_bucket_object_lock.test", "retention_days", "1"),
					resource.TestCheckNoResourceAttr("artesca_bucket_object_lock.test", "retention_years"),
				),
			},
			{
				Config: testAccAccountConfig(rAcct) + testAccBucketObjectLockConfig(rBucket, "COMPLIANCE", "retention_years = 1"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("artesca_bucket_object_lock.test", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("artesca_bucket.test", plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_bucket_object_lock.test", "retention_mode", "COMPLIANCE"),
					resource.TestCheckResourceAttr("artesca_bucket_object_lock.test", "retention_years", "1"),
					resource.TestCheckNoResourceAttr("artesca_bucket_object_lock.test", "retention_days"),
				),
			},
			{
				ResourceName:                         "artesca_bucket_object_lock.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateID(rBucket)),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "bucket_name",
			},
			{
				ResourceName:                         "artesca_bucket.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateID(rBucket)),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"force_destroy"},
			},
		},
	})
}

func TestAccBucketObjectLock_bucketWithoutObjectLock(t *testing.T) {
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
  account_name = artesca_account.test.name
  name         = %q
}

resource "artesca_bucket_object_lock" "test" {
  account_name   = artesca_account.test.name
  bucket_name    = artesca_bucket.test.name
  retention_mode = "GOVERNANCE"
  retention_days = 1
}
`, rBucket),
				ExpectError: regexp.MustCompile(`object_lock_enabled = true`),
			},
		},
	})
}

func testAccBucketObjectLockConfig(bucket, mode, period string) string {
	return fmt.Sprintf(`
resource "artesca_bucket" "test" {
  account_name        = artesca_account.test.name
  name                = %q
  object_lock_enabled = true
  force_destroy       = true
}

resource "artesca_bucket_object_lock" "test" {
  account_name   = artesca_account.test.name
  bucket_name    = artesca_bucket.test.name
  retention_mode = %q
  %s
}
`, bucket, mode, period)
}
