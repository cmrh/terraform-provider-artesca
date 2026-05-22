package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceLocations_basic(t *testing.T) {
	rName := randomName("tf-acc-ds-locs")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLocationSourceConfig(rName) + `
data "artesca_locations" "all" {
  depends_on = [artesca_location.source]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// At least one (the one we just created); built-in locations also appear.
					resource.TestCheckResourceAttrSet("data.artesca_locations.all", "locations.#"),
				),
			},
		},
	})
}
