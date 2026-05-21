package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicy_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rPolicy := randomName("tf-acc-mp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyConfig(rAcct, rPolicy),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_policy.test", "name", rPolicy),
					resource.TestCheckResourceAttrSet("artesca_policy.test", "arn"),
					resource.TestCheckResourceAttrSet("artesca_policy.test", "policy_id"),
				),
			},
		},
	})
}

func testAccPolicyConfig(accountName, policyName string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_policy" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = %q

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
`, policyName)
}
