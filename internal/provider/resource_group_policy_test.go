package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupPolicy_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rGroup := randomName("tf-acc-grp")
	rPolicy := randomName("tf-acc-pol")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupPolicyConfig(rAcct, rGroup, rPolicy),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_group_policy.test", "policy_name", rPolicy),
					resource.TestCheckResourceAttr("artesca_group_policy.test", "group_name", rGroup),
				),
			},
		},
	})
}

func TestAccGroupPolicy_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rGroup := randomName("tf-acc-grp")
	rPolicy := randomName("tf-acc-pol")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupPolicyDestroy,
		Steps: []resource.TestStep{
			{Config: testAccGroupPolicyConfig(rAcct, rGroup, rPolicy)},
			{
				ResourceName:                         "artesca_group_policy.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateID(rGroup + "/" + rPolicy)),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "policy_name",
				ImportStateVerifyIgnore:              []string{"policy_document"},
			},
		},
	})
}

func testAccGroupPolicyConfig(accountName, groupName, policyName string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_group" "test" {
  account_name = artesca_account.test.name
  name               = %q
}

resource "artesca_group_policy" "test" {
  account_name = artesca_account.test.name
  group_name         = artesca_group.test.name
  policy_name        = %q

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject"]
        Resource = "arn:aws:s3:::*"
      }
    ]
  })
}
`, groupName, policyName)
}
