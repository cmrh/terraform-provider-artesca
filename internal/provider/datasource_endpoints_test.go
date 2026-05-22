package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceEndpoints_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "artesca_endpoints" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Built-in endpoints exist on every cluster.
					resource.TestCheckResourceAttrSet("data.artesca_endpoints.all", "endpoints.#"),
				),
			},
		},
	})
}
