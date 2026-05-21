package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRolePolicyAttachment_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rRole := randomName("tf-acc-role")
	rPolicy := randomName("tf-acc-mp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRolePolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRolePolicyAttachmentConfig(rAcct, rRole, rPolicy),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_role_policy_attachment.test", "role_name", rRole),
					resource.TestCheckResourceAttrSet("artesca_role_policy_attachment.test", "policy_arn"),
				),
			},
		},
	})
}

func testAccRolePolicyAttachmentConfig(accountName, roleName, policyName string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_role" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = %q

  assume_role_policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [{ Effect = "Allow", Principal = { AWS = "*" }, Action = "sts:AssumeRole" }]
  })
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

resource "artesca_role_policy_attachment" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  role_name          = artesca_role.test.name
  policy_arn         = artesca_policy.test.arn
}
`, roleName, policyName)
}
