package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceResourceConfig("ssh", true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_service.test", "service", "ssh"),
					resource.TestCheckResourceAttr("truenas_service.test", "enable", "true"),
					resource.TestCheckResourceAttr("truenas_service.test", "running", "true"),
					resource.TestCheckResourceAttr("truenas_service.test", "state", "RUNNING"),
					resource.TestCheckResourceAttrSet("truenas_service.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_service.test",
				ImportState:       true,
				ImportStateId:     "ssh",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccServiceResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceResourceConfig("ssh", true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_service.test", "enable", "true"),
					resource.TestCheckResourceAttr("truenas_service.test", "running", "true"),
					resource.TestCheckResourceAttr("truenas_service.test", "state", "RUNNING"),
				),
			},
			{
				Config: testAccServiceResourceConfig("ssh", false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_service.test", "enable", "false"),
					resource.TestCheckResourceAttr("truenas_service.test", "running", "false"),
					resource.TestCheckResourceAttr("truenas_service.test", "state", "STOPPED"),
				),
			},
		},
	})
}

func testAccServiceResourceConfig(service string, enable bool, running bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_service" "test" {
  service = %q
  enable  = %t
  running = %t
}
`, service, enable, running)
}
