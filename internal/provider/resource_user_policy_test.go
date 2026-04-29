package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserPolicy_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserPolicyConfig(rAcct, rUser, "s3:GetObject"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_user_policy.test", "username", rUser),
					resource.TestCheckResourceAttr("artesca_user_policy.test", "policy_name", "tf-acc-policy"),
					resource.TestCheckResourceAttrSet("artesca_user_policy.test", "policy_document"),
				),
			},
		},
	})
}

func TestAccUserPolicy_update(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserPolicyConfig(rAcct, rUser, "s3:GetObject"),
			},
			{
				Config: testAccUserPolicyConfig(rAcct, rUser, "s3:*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("artesca_user_policy.test", "policy_document"),
				),
			},
		},
	})
}

func testAccUserPolicyConfig(accountName, username, action string) string {
	return testAccUserConfig(accountName, username) + fmt.Sprintf(`
resource "artesca_user_policy" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = artesca_user.test.username
  policy_name        = "tf-acc-policy"

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = [%q]
        Resource = "arn:aws:s3:::*"
      }
    ]
  })
}
`, action)
}
