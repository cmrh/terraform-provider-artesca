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

func testAccGroupMembershipConfig(accountName, username, groupName string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_user" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = %q
}

resource "artesca_group" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = %q
}

resource "artesca_group_membership" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  group_name         = artesca_group.test.name
  username           = artesca_user.test.username
}
`, username, groupName)
}
