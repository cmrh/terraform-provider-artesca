package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUser_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig(rAcct, rUser),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_user.test", "username", rUser),
					resource.TestCheckResourceAttrSet("artesca_user.test", "user_id"),
					resource.TestCheckResourceAttrSet("artesca_user.test", "arn"),
				),
			},
		},
	})
}

func TestAccUser_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{Config: testAccUserConfig(rAcct, rUser)},
			{
				ResourceName:                         "artesca_user.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateID(rUser)),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "username",
			},
		},
	})
}

func testAccUserConfig(accountName, username string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_user" "test" {
  account_name = artesca_account.test.name
  username           = %q
}
`, username)
}
