package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNVMeTPortResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTPortResourceConfig("0.0.0.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_port.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_port.test", "index"),
					resource.TestCheckResourceAttr("truenas_nvmet_port.test", "addr_trtype", "TCP"),
					resource.TestCheckResourceAttr("truenas_nvmet_port.test", "addr_traddr", "0.0.0.0"),
					resource.TestCheckResourceAttr("truenas_nvmet_port.test", "addr_trsvcid", "4420"),
					resource.TestCheckResourceAttr("truenas_nvmet_port.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_port.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNVMeTPortResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTPortResourceConfig("0.0.0.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_port.test", "enabled", "true"),
				),
			},
			{
				Config: testAccNVMeTPortResourceConfigDisabled("0.0.0.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_port.test", "enabled", "false"),
				),
			},
		},
	})
}

func testAccNVMeTPortResourceConfig(addr string) string {
	return testAccProviderConfig() + `
resource "truenas_nvmet_port" "test" {
  addr_trtype  = "TCP"
  addr_traddr  = "` + addr + `"
  addr_trsvcid = 4420
}
`
}

func testAccNVMeTPortResourceConfigDisabled(addr string) string {
	return testAccProviderConfig() + `
resource "truenas_nvmet_port" "test" {
  addr_trtype  = "TCP"
  addr_traddr  = "` + addr + `"
  addr_trsvcid = 4420
  enabled      = false
}
`
}
