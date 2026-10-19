package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
				ImportStateIdFunc:                    testAccImportWithAccount(testAccImportStateAttr("artesca_policy.test", "arn")),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "arn",
				ImportStateVerifyIgnore:              []string{"policy_document"},
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
  account_name = artesca_account.test.name
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

// Changing the document creates a new default version in place, keeping the
// policy and its attachments. Six changes go past ARTESCA's 5-version limit,
// and destroy deletes a policy that has several versions.
func TestAccPolicy_updateDocumentInPlace(t *testing.T) {
	rAcct, rUser, rPolicy := randomName("tf-acc"), randomName("tf-acc-user"), randomName("tf-acc-pol")
	config := func(sid int) string {
		return testAccAccountConfig(rAcct) + fmt.Sprintf(`
resource "artesca_user" "test" {
  account_name = artesca_account.test.name
  username     = %q
}

resource "artesca_policy" "test" {
  account_name = artesca_account.test.name
  name         = %q
  policy_document = jsonencode({
    Version   = "2012-10-17"
    Statement = [{ Sid = "S%d", Effect = "Allow", Action = "s3:GetObject", Resource = "*" }]
  })
}

resource "artesca_user_policy_attachment" "test" {
  account_name = artesca_account.test.name
  username     = artesca_user.test.username
  policy_arn   = artesca_policy.test.arn
}
`, rUser, rPolicy, sid)
	}

	steps := []resource.TestStep{{Config: config(0)}}
	for sid := 1; sid <= 6; sid++ {
		steps = append(steps, resource.TestStep{
			Config: config(sid),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("artesca_policy.test", plancheck.ResourceActionUpdate),
					plancheck.ExpectResourceAction("artesca_user_policy_attachment.test", plancheck.ResourceActionNoop),
				},
			},
			Check: resource.TestCheckResourceAttr("artesca_policy.test", "default_version_id", fmt.Sprintf("v%d", sid+1)),
		})
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}
