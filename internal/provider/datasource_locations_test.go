package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
					testAccCheckLocationsSorted("data.artesca_locations.all"),
				),
			},
		},
	})
}

func testAccCheckLocationsSorted(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		attrs := s.RootModule().Resources[name].Primary.Attributes
		n, err := strconv.Atoi(attrs["locations.#"])
		if err != nil {
			return err
		}
		if n < 2 {
			return fmt.Errorf("want at least 2 locations to check the order, got %d", n)
		}
		for i := 1; i < n; i++ {
			prev, cur := attrs[fmt.Sprintf("locations.%d.name", i-1)], attrs[fmt.Sprintf("locations.%d.name", i)]
			if prev > cur {
				return fmt.Errorf("locations not sorted by name: %q before %q", prev, cur)
			}
		}
		return nil
	}
}
