package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupPolicyAttachment_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rGroup := randomName("tf-acc-grp")
	rPolicy := randomName("tf-acc-mp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGroupPolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupPolicyAttachmentConfig(rAcct, rGroup, rPolicy),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_group_policy_attachment.test", "group_name", rGroup),
					resource.TestCheckResourceAttrSet("artesca_group_policy_attachment.test", "policy_arn"),
				),
			},
		},
	})
}

func testAccGroupPolicyAttachmentConfig(accountName, groupName, policyName string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_group" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = %q
}

resource "artesca_policy" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = %q

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [{ Effect = "Allow", Action = "s3:GetObject", Resource = "*" }]
  })
}

resource "artesca_group_policy_attachment" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  group_name         = artesca_group.test.name
  policy_arn         = artesca_policy.test.arn
}
`, groupName, policyName)
}
