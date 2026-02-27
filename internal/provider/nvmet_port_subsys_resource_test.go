package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNVMeTPortSubsysResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTPortSubsysResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_port_subsys.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_port_subsys.test", "port_id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_port_subsys.test", "subsys_id"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_port_subsys.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccNVMeTPortSubsysResourceConfig() string {
	return testAccProviderConfig() + `
resource "truenas_nvmet_port" "ps_test" {
  addr_trtype  = "TCP"
  addr_traddr  = "0.0.0.0"
  addr_trsvcid = 4421
}

resource "truenas_nvmet_subsys" "ps_test" {
  name           = "tf-acc-test-ps-subsys"
  allow_any_host = true
}

resource "truenas_nvmet_port_subsys" "test" {
  port_id   = truenas_nvmet_port.ps_test.id
  subsys_id = truenas_nvmet_subsys.ps_test.id
}
`
}
