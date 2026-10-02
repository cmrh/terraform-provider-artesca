package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
