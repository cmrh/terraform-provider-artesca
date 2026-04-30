package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEndpoint_basic(t *testing.T) {
	rLocName := randomName("tf-acc-loc")
	rHostname := fmt.Sprintf("%s.s3.test.example.com", randomName("tf-acc"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEndpointConfig(rLocName, rHostname),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_endpoint.test", "hostname", rHostname),
					resource.TestCheckResourceAttr("artesca_endpoint.test", "location_name", rLocName),
				),
			},
		},
	})
}

func TestAccEndpoint_importState(t *testing.T) {
	rLocName := randomName("tf-acc-loc")
	rHostname := fmt.Sprintf("%s.s3.test.example.com", randomName("tf-acc"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEndpointConfig(rLocName, rHostname),
			},
			{
				ResourceName:                         "artesca_endpoint.test",
				ImportState:                          true,
				ImportStateId:                        rHostname,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "hostname",
			},
		},
	})
}

func testAccEndpointConfig(locationName, hostname string) string {
	return testAccLocationSourceConfig(locationName) + fmt.Sprintf(`
resource "artesca_endpoint" "test" {
  hostname      = %q
  location_name = artesca_location.source.name
}
`, hostname)
}
