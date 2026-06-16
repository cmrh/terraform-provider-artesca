package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserPolicyAttachment_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-user")
	rPolicy := randomName("tf-acc-mp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserPolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccUserPolicyAttachmentConfig(rAcct, rUser, rPolicy),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_user_policy_attachment.test", "username", rUser),
					resource.TestCheckResourceAttrSet("artesca_user_policy_attachment.test", "policy_arn"),
				),
			},
		},
	})
}

func TestAccUserPolicyAttachment_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-user")
	rPolicy := randomName("tf-acc-mp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserPolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			{Config: testAccUserPolicyAttachmentConfig(rAcct, rUser, rPolicy)},
			{
				ResourceName:                         "artesca_user_policy_attachment.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportStateUserPolicyArn("artesca_user_policy_attachment.test"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "policy_arn",
				ImportStateVerifyIgnore:              []string{"account_access_key", "account_secret_key"},
			},
		},
	})
}

func testAccImportStateUserPolicyArn(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found", resourceName)
		}
		return rs.Primary.Attributes["username"] + "/" + rs.Primary.Attributes["policy_arn"], nil
	}
}

func testAccUserPolicyAttachmentConfig(accountName, username, policyName string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_user" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = %q
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

resource "artesca_user_policy_attachment" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = artesca_user.test.username
  policy_arn         = artesca_policy.test.arn
}
`, username, policyName)
}
