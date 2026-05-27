package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceInstance_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "artesca_instance" "current" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.artesca_instance.current", "instance_id"),
					resource.TestCheckResourceAttrSet("data.artesca_instance.current", "state"),
					resource.TestCheckResourceAttrSet("data.artesca_instance.current", "public_key"),
					resource.TestCheckResourceAttrSet("data.artesca_instance.current", "ip_address"),
					resource.TestCheckResourceAttrSet("data.artesca_instance.current", "last_seen"),
					resource.TestCheckResourceAttrSet("data.artesca_instance.current", "running_configuration_version"),
					resource.TestCheckResourceAttrSet("data.artesca_instance.current", "server_version"),
					resource.TestMatchResourceAttr("data.artesca_instance.current",
						"instance_id", regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)),
				),
			},
		},
	})
}

func TestAccDataSourceInstance_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "artesca_instance" "bad" {
  instance_id = "00000000-0000-0000-0000-000000000000"
}
`,
				ExpectError: regexp.MustCompile(`(?s)Error reading instance`),
			},
		},
	})
}
