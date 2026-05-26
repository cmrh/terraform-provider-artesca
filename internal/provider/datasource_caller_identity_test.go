package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceCallerIdentity_basic(t *testing.T) {
	rAcct := randomName("tf-acc-ci")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) + `
data "artesca_caller_identity" "current" {
  access_key = artesca_account.test.access_key
  secret_key = artesca_account.test.secret_key
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.artesca_caller_identity.current", "user_id"),
					resource.TestCheckResourceAttrSet("data.artesca_caller_identity.current", "account"),
					resource.TestCheckResourceAttrSet("data.artesca_caller_identity.current", "arn"),
					resource.TestCheckResourceAttrPair(
						"data.artesca_caller_identity.current", "account",
						"artesca_account.test", "id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceCallerIdentity_invalidCreds(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "artesca_caller_identity" "bad" {
  access_key = "AKIAINVALIDKEYXXXXXX"
  secret_key = "invalidsecretxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
`,
				ExpectError: regexp.MustCompile(`(?s)(STS GetCallerIdentity|InvalidClientTokenId|SignatureDoesNotMatch|get caller identity)`),
			},
		},
	})
}
