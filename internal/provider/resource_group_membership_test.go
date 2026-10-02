package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupMembership_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-user")
	rGroup := randomName("tf-acc-grp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupMembershipDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupMembershipConfig(rAcct, rUser, rGroup),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_group_membership.test", "group_name", rGroup),
					resource.TestCheckResourceAttr("artesca_group_membership.test", "username", rUser),
				),
			},
		},
	})
}

func TestAccGroupMembership_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-user")
	rGroup := randomName("tf-acc-grp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupMembershipDestroy,
		Steps: []resource.TestStep{
			{Config: testAccGroupMembershipConfig(rAcct, rUser, rGroup)},
			{
				ResourceName:                         "artesca_group_membership.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateID(rGroup + "/" + rUser)),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "username",
			},
		},
	})
}

func testAccGroupMembershipConfig(accountName, username, groupName string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_user" "test" {
  account_name = artesca_account.test.name
  username           = %q
}

resource "artesca_group" "test" {
  account_name = artesca_account.test.name
  name               = %q
}

resource "artesca_group_membership" "test" {
  account_name = artesca_account.test.name
  group_name         = artesca_group.test.name
  username           = artesca_user.test.username
}
`, username, groupName)
}
