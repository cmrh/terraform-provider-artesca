package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceAccounts_basic(t *testing.T) {
	rName := randomName("tf-acc-ds-accts")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rName) + `
data "artesca_accounts" "all" {
  depends_on = [artesca_account.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// At least one account exists (the one we created); list never empty.
					resource.TestCheckResourceAttrSet("data.artesca_accounts.all", "accounts.#"),
				),
			},
		},
	})
}
