package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccAccount_basic(t *testing.T) {
	rName := randomName("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_account.test", "name", rName),
					resource.TestCheckResourceAttr("artesca_account.test", "email", rName+"@test.example.com"),
					resource.TestCheckResourceAttrSet("artesca_account.test", "access_key"),
					resource.TestCheckResourceAttrSet("artesca_account.test", "secret_key"),
					resource.TestMatchResourceAttr("artesca_account.test", "arn", regexp.MustCompile(`^arn:aws:iam::\d{12}:root$`)),
					resource.TestCheckResourceAttrSet("artesca_account.test", "canonical_id"),
					resource.TestCheckResourceAttrSet("artesca_account.test", "id"),
				),
			},
		},
	})
}

func TestAccAccount_importState(t *testing.T) {
	rName := randomName("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rName),
			},
			{
				ResourceName:      "artesca_account.test",
				ImportState:       true,
				ImportStateId:     rName,
				ImportStateVerify: true,
				// Email and keys are not readable from the account listing.
				ImportStateVerifyIgnore: []string{"access_key", "secret_key", "email"},
			},
		},
	})
}

func testAccAccountEmailConfig(name, email string) string {
	if email == "" {
		return fmt.Sprintf(`
resource "artesca_account" "test" {
  name = %q
}
`, name)
	}
	return fmt.Sprintf(`
resource "artesca_account" "test" {
  name  = %q
  email = %q
}
`, name, email)
}

func TestAccAccount_emailRemovedKeepsAccount(t *testing.T) {
	rName := randomName("tf-acc")
	email := rName + "@test.example.com"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccAccountEmailConfig(rName, email)},
			{
				Config: testAccAccountEmailConfig(rName, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("artesca_account.test", plancheck.ResourceActionNoop)},
				},
				Check: resource.TestCheckResourceAttr("artesca_account.test", "email", email),
			},
		},
	})
}

func TestAccAccount_emailRequiredOnCreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAccountEmailConfig(randomName("tf-acc"), ""),
				ExpectError: regexp.MustCompile(`email is required to create an account`),
			},
		},
	})
}

// An account imported with an import block has no email in state; the
// configured email is adopted in place. The seed account is handed over with
// a removed block so only one resource manages it.
func TestAccAccount_emailAdoptedAfterImport(t *testing.T) {
	rName := randomName("tf-acc")
	email := rName + "@test.example.com"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "artesca_account" "seed" {
  name  = %q
  email = %q
}
`, rName, email),
			},
			{
				Config: fmt.Sprintf(`
removed {
  from = artesca_account.seed
  lifecycle {
    destroy = false
  }
}

import {
  to = artesca_account.test
  id = %q
}

resource "artesca_account" "test" {
  name  = %q
  email = %q
}
`, rName, rName, email),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("artesca_account.test", plancheck.ResourceActionUpdate)},
				},
				Check: resource.TestCheckResourceAttr("artesca_account.test", "email", email),
			},
		},
	})
}
