package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSIInitiatorResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIInitiatorResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_initiator.test", "id"),
					resource.TestCheckResourceAttr("truenas_iscsi_initiator.test", "comment", "test initiator"),
				),
			},
			{
				ResourceName:      "truenas_iscsi_initiator.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccISCSIInitiatorResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIInitiatorResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_initiator.test", "comment", "test initiator"),
				),
			},
			{
				Config: testAccISCSIInitiatorResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_initiator.test", "comment", "updated initiator"),
					resource.TestCheckResourceAttr("truenas_iscsi_initiator.test", "initiators.#", "1"),
				),
			},
		},
	})
}

func testAccISCSIInitiatorResourceConfig() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_initiator" "test" {
  comment = "test initiator"
}
`
}

func testAccISCSIInitiatorResourceConfigUpdated() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_initiator" "test" {
  initiators = ["iqn.2025-01.com.example:test"]
  comment    = "updated initiator"
}
`
}
