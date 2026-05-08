package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceLocation_basic(t *testing.T) {
	rName := randomName("tf-acc-ds-loc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLocationSourceConfig(rName) + `
data "artesca_location" "lookup" {
  name = artesca_location.source.name
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artesca_location.lookup", "name", rName),
					resource.TestCheckResourceAttr("data.artesca_location.lookup", "location_type", "location-scality-ring-s3-v1"),
					resource.TestCheckResourceAttr("data.artesca_location.lookup", "is_builtin", "false"),
					resource.TestCheckResourceAttrSet("data.artesca_location.lookup", "details.endpoint"),
					resource.TestCheckResourceAttrSet("data.artesca_location.lookup", "details.bucket_name"),
				),
			},
		},
	})
}

func TestAccDataSourceLocation_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "artesca_location" "missing" {
  name = "tf-acc-no-such-location-12345"
}
`,
				ExpectError: regexp.MustCompile(`(?s)Location not found`),
			},
		},
	})
}
