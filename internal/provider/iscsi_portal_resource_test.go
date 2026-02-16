package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSIPortalResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIPortalResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_portal.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_iscsi_portal.test", "tag"),
					resource.TestCheckResourceAttr("truenas_iscsi_portal.test", "listen.#", "1"),
					resource.TestCheckResourceAttr("truenas_iscsi_portal.test", "listen.0.ip", "0.0.0.0"),
				),
			},
			{
				ResourceName:      "truenas_iscsi_portal.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccISCSIPortalResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIPortalResourceConfigWithComment(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_portal.test", "comment", "test portal"),
				),
			},
			{
				Config: testAccISCSIPortalResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_portal.test", "comment", "updated portal"),
				),
			},
		},
	})
}

func testAccISCSIPortalResourceConfig() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_portal" "test" {
  listen {
    ip = "0.0.0.0"
  }
}
`
}

func testAccISCSIPortalResourceConfigWithComment() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_portal" "test" {
  listen {
    ip = "0.0.0.0"
  }
  comment = "test portal"
}
`
}

func testAccISCSIPortalResourceConfigUpdated() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_portal" "test" {
  listen {
    ip = "0.0.0.0"
  }
  comment = "updated portal"
}
`
}
