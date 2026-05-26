package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceUser_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-ds-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig(rAcct, rUser) + `
data "artesca_user" "lookup" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = artesca_user.test.username
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artesca_user.lookup", "username", rUser),
					resource.TestCheckResourceAttrSet("data.artesca_user.lookup", "user_id"),
					resource.TestCheckResourceAttrSet("data.artesca_user.lookup", "arn"),
					resource.TestCheckResourceAttrPair(
						"data.artesca_user.lookup", "user_id",
						"artesca_user.test", "user_id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceUser_notFound(t *testing.T) {
	rAcct := randomName("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) + `
data "artesca_user" "missing" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = "tf-acc-no-such-user-12345"
}
`,
				ExpectError: regexp.MustCompile(`(?s)IAM user not found`),
			},
		},
	})
}
