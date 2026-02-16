package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSITargetResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSITargetResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_target.test", "id"),
					resource.TestCheckResourceAttr("truenas_iscsi_target.test", "name", "tf-acc-test-target"),
					resource.TestCheckResourceAttr("truenas_iscsi_target.test", "groups.#", "1"),
				),
			},
			{
				ResourceName:      "truenas_iscsi_target.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccISCSITargetResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSITargetResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_target.test", "name", "tf-acc-test-target"),
				),
			},
			{
				Config: testAccISCSITargetResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_target.test", "alias", "updated alias"),
				),
			},
		},
	})
}

func testAccISCSITargetResourceConfig() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_portal" "test_target" {
  listen {
    ip = "0.0.0.0"
  }
}

resource "truenas_iscsi_target" "test" {
  name = "tf-acc-test-target"
  groups {
    portal = truenas_iscsi_portal.test_target.id
  }
}
`
}

func testAccISCSITargetResourceConfigUpdated() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_portal" "test_target" {
  listen {
    ip = "0.0.0.0"
  }
}

resource "truenas_iscsi_target" "test" {
  name  = "tf-acc-test-target"
  alias = "updated alias"
  groups {
    portal = truenas_iscsi_portal.test_target.id
  }
}
`
}
