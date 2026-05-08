package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceAccount_basic(t *testing.T) {
	rName := randomName("tf-acc-ds-acct")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rName) + `
data "artesca_account" "lookup" {
  name = artesca_account.test.name
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artesca_account.lookup", "name", rName),
					resource.TestCheckResourceAttrSet("data.artesca_account.lookup", "id"),
					resource.TestCheckResourceAttrSet("data.artesca_account.lookup", "canonical_id"),
					resource.TestCheckResourceAttrSet("data.artesca_account.lookup", "arn"),
					resource.TestCheckResourceAttrPair(
						"data.artesca_account.lookup", "id",
						"artesca_account.test", "id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceAccount_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "artesca_account" "missing" {
  name = "tf-acc-no-such-account-12345"
}
`,
				ExpectError: regexp.MustCompile(`(?s)Account not found`),
			},
		},
	})
}
