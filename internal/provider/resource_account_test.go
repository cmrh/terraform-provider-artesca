package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAccount_basic(t *testing.T) {
	rName := randomName("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_account.test", "name", rName),
					resource.TestCheckResourceAttr("artesca_account.test", "email", rName+"@test.example.com"),
					resource.TestCheckResourceAttrSet("artesca_account.test", "access_key"),
					resource.TestCheckResourceAttrSet("artesca_account.test", "secret_key"),
					resource.TestCheckResourceAttrSet("artesca_account.test", "arn"),
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
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rName),
			},
			{
				ResourceName:            "artesca_account.test",
				ImportState:             true,
				ImportStateId:           rName,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"access_key", "secret_key"},
			},
		},
	})
}
