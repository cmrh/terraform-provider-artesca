package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Documents built by artesca_policy_document are accepted by ARTESCA as a
// managed policy and a bucket policy, and plan nothing after apply.
func TestAccDataSourcePolicyDocument_basic(t *testing.T) {
	rAcct, rBucket, rPolicy := randomName("tf-acc"), randomName("tf-acc-bkt"), randomName("tf-acc-pol")
	config := testAccAccountConfig(rAcct) + fmt.Sprintf(`
resource "artesca_bucket" "test" {
  account_name = artesca_account.test.name
  name         = %[1]q
}

data "artesca_policy_document" "read" {
  statement {
    sid       = "Read"
    actions   = ["s3:GetObject", "s3:ListBucket"]
    resources = ["arn:aws:s3:::${artesca_bucket.test.name}", "arn:aws:s3:::${artesca_bucket.test.name}/*"]
  }
}

resource "artesca_policy" "test" {
  account_name    = artesca_account.test.name
  name            = %[2]q
  policy_document = data.artesca_policy_document.read.json
}

data "artesca_policy_document" "bucket" {
  statement {
    sid       = "PublicRead"
    actions   = ["s3:GetObject"]
    resources = ["arn:aws:s3:::${artesca_bucket.test.name}/*"]
    principals {
      type        = "*"
      identifiers = ["*"]
    }
  }
}

resource "artesca_bucket_policy" "test" {
  account_name = artesca_account.test.name
  bucket_name  = artesca_bucket.test.name
  policy       = data.artesca_policy_document.bucket.json
}
`, rBucket, rPolicy)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("artesca_policy.test", "policy_document", "data.artesca_policy_document.read", "json"),
					resource.TestCheckResourceAttrSet("artesca_bucket_policy.test", "policy"),
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
