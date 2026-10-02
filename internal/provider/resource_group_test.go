package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroup_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rGroup := randomName("tf-acc-grp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfig(rAcct, rGroup),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_group.test", "name", rGroup),
					resource.TestCheckResourceAttrSet("artesca_group.test", "group_id"),
					resource.TestCheckResourceAttrSet("artesca_group.test", "arn"),
				),
			},
		},
	})
}

func TestAccGroup_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rGroup := randomName("tf-acc-grp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupDestroy,
		Steps: []resource.TestStep{
			{Config: testAccGroupConfig(rAcct, rGroup)},
			{
				ResourceName:                         "artesca_group.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateID(rGroup)),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func testAccGroupConfig(accountName, groupName string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_group" "test" {
  account_name = artesca_account.test.name
  name               = %q
}
`, groupName)
}
