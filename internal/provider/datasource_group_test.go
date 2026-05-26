package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceGroup_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rGroup := randomName("tf-acc-ds-group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfig(rAcct, rGroup) + `
data "artesca_group" "lookup" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = artesca_group.test.name
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artesca_group.lookup", "name", rGroup),
					resource.TestCheckResourceAttrSet("data.artesca_group.lookup", "group_id"),
					resource.TestCheckResourceAttrSet("data.artesca_group.lookup", "arn"),
					resource.TestCheckResourceAttrPair(
						"data.artesca_group.lookup", "group_id",
						"artesca_group.test", "group_id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceGroup_notFound(t *testing.T) {
	rAcct := randomName("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) + `
data "artesca_group" "missing" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = "tf-acc-no-such-group-12345"
}
`,
				ExpectError: regexp.MustCompile(`(?s)IAM group not found`),
			},
		},
	})
}
