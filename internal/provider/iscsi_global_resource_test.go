package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSIGlobalResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIGlobalResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_global.test", "id"),
					resource.TestCheckResourceAttr("truenas_iscsi_global.test", "basename", "iqn.2005-10.org.freenas.ctl"),
					resource.TestCheckResourceAttrSet("truenas_iscsi_global.test", "listen_port"),
				),
			},
			{
				ResourceName:      "truenas_iscsi_global.test",
				ImportState:       true,
				ImportStateId:     "iscsi-global",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccISCSIGlobalResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIGlobalResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_global.test", "basename", "iqn.2005-10.org.freenas.ctl"),
				),
			},
			{
				Config: testAccISCSIGlobalResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_global.test", "basename", "iqn.2025-01.com.example:storage"),
				),
			},
		},
	})
}

func testAccISCSIGlobalResourceConfig() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_global" "test" {
  basename = "iqn.2005-10.org.freenas.ctl"
}
`
}

func testAccISCSIGlobalResourceConfigUpdated() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_global" "test" {
  basename = "iqn.2025-01.com.example:storage"
}
`
}
