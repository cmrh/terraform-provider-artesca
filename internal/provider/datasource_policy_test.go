package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourcePolicy_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rPolicy := randomName("tf-acc-ds-mp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyConfig(rAcct, rPolicy) + `
data "artesca_policy" "lookup" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  arn                = artesca_policy.test.arn
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artesca_policy.lookup", "name", rPolicy),
					resource.TestCheckResourceAttrSet("data.artesca_policy.lookup", "policy_id"),
					resource.TestCheckResourceAttrSet("data.artesca_policy.lookup", "default_version_id"),
					resource.TestCheckResourceAttrSet("data.artesca_policy.lookup", "policy_document"),
					resource.TestCheckResourceAttrPair(
						"data.artesca_policy.lookup", "policy_id",
						"artesca_policy.test", "policy_id",
					),
				),
			},
		},
	})
}

func TestAccDataSourcePolicy_notFound(t *testing.T) {
	rAcct := randomName("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) + `
data "artesca_policy" "missing" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  arn                = "arn:aws:iam::000000000000:policy/no-such-policy"
}
`,
				ExpectError: regexp.MustCompile(`(?s)(IAM policy not found|Error reading IAM policy)`),
			},
		},
	})
}
