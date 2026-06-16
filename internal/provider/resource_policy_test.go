package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

func TestAccPolicy_importState(t *testing.T) {
	rAcct := randomName("tf-acc")
	rPolicy := randomName("tf-acc-mp")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPolicyDestroy,
		Steps: []resource.TestStep{
			{Config: testAccPolicyConfig(rAcct, rPolicy)},
			{
				ResourceName:                         "artesca_policy.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportStateAttr("artesca_policy.test", "arn"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "arn",
				ImportStateVerifyIgnore:              []string{"account_access_key", "account_secret_key", "policy_document"},
			},
		},
	})
}

// testAccImportStateAttr returns an ImportStateIdFunc that pulls the named
// attribute out of the named resource's state.
func testAccImportStateAttr(resourceName, attr string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found", resourceName)
		}
		v := rs.Primary.Attributes[attr]
		if v == "" {
			return "", fmt.Errorf("%s.%s is empty", resourceName, attr)
		}
		return v, nil
	}
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
