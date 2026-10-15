package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccBucket_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
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

func TestAccBucket_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBucket, "artesca_location.source.name", false),
			},
			{
				ResourceName:                         "artesca_bucket.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateID(rBucket)),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
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
		CheckDestroy:             testAccCheckBucketDestroy,
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

func TestAccBucket_alreadyOwned(t *testing.T) {
	rAcct := randomName("tf-acc")
	rBucket := randomName("tf-acc-bkt")
	config := testAccAccountConfig(rAcct) + testAccBucketConfig("test", rBucket, "null", false)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{Config: config},
			{
				Config:      config + testAccBucketConfig("duplicate", rBucket, "null", false),
				ExpectError: regexp.MustCompile(`already owns bucket`),
			},
		},
	})
}

// HEAD on another account's bucket returns 403, which must surface as an
// error rather than read as "bucket absent".
func TestAccBucket_forbiddenIsError(t *testing.T) {
	rAcct, rOther := randomName("tf-acc"), randomName("tf-acc")
	rBucket := randomName("tf-acc-bkt")
	config := testAccAccountConfig(rAcct) + testAccBucketConfig("test", rBucket, "null", false) + fmt.Sprintf(`
resource "artesca_account" "other" {
  name  = %q
  email = "%s@test.example.com"
}
`, rOther, rOther)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{Config: config},
			{
				Config: config + fmt.Sprintf(`
import {
  to = artesca_bucket.other
  id = "%s/%s"
}

resource "artesca_bucket" "other" {
  name         = %q
  account_name = artesca_account.other.name
}
`, rOther, rBucket, rBucket),
				ExpectError: regexp.MustCompile(`access denied`),
			},
		},
	})
}

// Without versioning_enabled in the configuration, versioning is not managed:
// a new bucket reports its state, and removing the attribute from a versioned
// bucket's configuration leaves versioning enabled.
func TestAccBucket_versioningUnmanaged(t *testing.T) {
	rAcct := randomName("tf-acc")
	rBucket := randomName("tf-acc-bkt")
	unset := testAccAccountConfig(rAcct) + fmt.Sprintf(`
resource "artesca_bucket" "test" {
  name         = %q
  account_name = artesca_account.test.name
}
`, rBucket)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: unset,
				Check:  resource.TestCheckResourceAttr("artesca_bucket.test", "versioning_enabled", "false"),
			},
			{
				Config: testAccAccountConfig(rAcct) + testAccBucketConfig("test", rBucket, "null", true),
				Check:  resource.TestCheckResourceAttr("artesca_bucket.test", "versioning_enabled", "true"),
			},
			{
				Config: unset,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("artesca_bucket.test", plancheck.ResourceActionNoop)},
				},
				Check: resource.TestCheckResourceAttr("artesca_bucket.test", "versioning_enabled", "true"),
			},
			{
				// Refresh reads the bucket: versioning is still enabled.
				Config:   unset,
				PlanOnly: true,
			},
		},
	})
}
