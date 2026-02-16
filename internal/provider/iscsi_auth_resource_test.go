package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSIAuthResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIAuthResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_auth.test", "id"),
					resource.TestCheckResourceAttr("truenas_iscsi_auth.test", "tag", "1"),
					resource.TestCheckResourceAttr("truenas_iscsi_auth.test", "user", "testchapuser"),
				),
			},
			{
				ResourceName:            "truenas_iscsi_auth.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret", "peersecret"},
			},
		},
	})
}

func TestAccISCSIAuthResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIAuthResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_auth.test", "user", "testchapuser"),
				),
			},
			{
				Config: testAccISCSIAuthResourceConfigUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_auth.test", "user", "updatedchapuser"),
				),
			},
		},
	})
}

func testAccISCSIAuthResourceConfig() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_auth" "test" {
  tag    = 1
  user   = "testchapuser"
  secret = "abcdef123456"
}
`
}

func testAccISCSIAuthResourceConfigUpdated() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_auth" "test" {
  tag    = 1
  user   = "updatedchapuser"
  secret = "abcdef654321"
}
`
}
