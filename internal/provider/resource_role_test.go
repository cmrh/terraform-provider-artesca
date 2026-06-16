package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRole_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rRole := randomName("tf-acc-role")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig(rAcct, rRole),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_role.test", "name", rRole),
					resource.TestCheckResourceAttrSet("artesca_role.test", "role_id"),
					resource.TestCheckResourceAttrSet("artesca_role.test", "arn"),
				),
			},
		},
	})
}

func TestAccRole_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rRole := randomName("tf-acc-role")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			{Config: testAccRoleConfig(rAcct, rRole)},
			{
				ResourceName:                         "artesca_role.test",
				ImportState:                          true,
				ImportStateId:                        rRole,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"account_access_key", "account_secret_key", "assume_role_policy_document"},
			},
		},
	})
}

func testAccRoleConfig(accountName, roleName string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_role" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = %q

  assume_role_policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = "*" }
        Action    = "sts:AssumeRole"
      }
    ]
  })
}
`, roleName)
}
