package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNVMeTGlobalResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTGlobalResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_global.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_global.test", "basenqn"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_global.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNVMeTGlobalResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTGlobalResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_global.test", "id"),
				),
			},
			{
				Config: testAccNVMeTGlobalResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_global.test", "ana", "false"),
				),
			},
		},
	})
}

func testAccNVMeTGlobalResourceConfig() string {
	return testAccProviderConfig() + `
resource "truenas_nvmet_global" "test" {
}
`
}

func testAccNVMeTGlobalResourceConfigUpdated() string {
	return testAccProviderConfig() + `
resource "truenas_nvmet_global" "test" {
  ana = false
}
`
}
