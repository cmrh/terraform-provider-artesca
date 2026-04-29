package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserAccessKey_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserAccessKeyConfig(rAcct, rUser),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_user_access_key.test", "username", rUser),
					resource.TestCheckResourceAttrSet("artesca_user_access_key.test", "access_key_id"),
					resource.TestCheckResourceAttrSet("artesca_user_access_key.test", "secret_access_key"),
					resource.TestCheckResourceAttr("artesca_user_access_key.test", "status", "Active"),
				),
			},
		},
	})
}

func testAccUserAccessKeyConfig(accountName, username string) string {
	return testAccUserConfig(accountName, username) + `
resource "artesca_user_access_key" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = artesca_user.test.username
}
`
}
