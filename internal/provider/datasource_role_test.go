package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceRole_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rRole := randomName("tf-acc-ds-role")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig(rAcct, rRole) + `
data "artesca_role" "lookup" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = artesca_role.test.name
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artesca_role.lookup", "name", rRole),
					resource.TestCheckResourceAttrSet("data.artesca_role.lookup", "role_id"),
					resource.TestCheckResourceAttrSet("data.artesca_role.lookup", "arn"),
					resource.TestCheckResourceAttrSet("data.artesca_role.lookup", "assume_role_policy_document"),
					resource.TestCheckResourceAttrPair(
						"data.artesca_role.lookup", "role_id",
						"artesca_role.test", "role_id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceRole_notFound(t *testing.T) {
	rAcct := randomName("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) + `
data "artesca_role" "missing" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = "tf-acc-no-such-role-12345"
}
`,
				ExpectError: regexp.MustCompile(`(?s)IAM role not found`),
			},
		},
	})
}
