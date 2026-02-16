package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSIExtentResource_basic(t *testing.T) {
	pool := testAccPoolName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIExtentResourceConfig(pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_extent.test", "id"),
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "name", "tf-acc-test-extent"),
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "type", "FILE"),
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("truenas_iscsi_extent.test", "naa"),
					resource.TestCheckResourceAttrSet("truenas_iscsi_extent.test", "serial"),
				),
			},
			{
				ResourceName:      "truenas_iscsi_extent.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccISCSIExtentResource_update(t *testing.T) {
	pool := testAccPoolName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIExtentResourceConfig(pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "name", "tf-acc-test-extent"),
				),
			},
			{
				Config: testAccISCSIExtentResourceConfigUpdated(pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_iscsi_extent.test", "comment", "updated extent"),
				),
			},
		},
	})
}

func testAccISCSIExtentResourceConfig(pool string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_iscsi_extent" "test" {
  name     = "tf-acc-test-extent"
  type     = "FILE"
  path     = "/mnt/%s/iscsi-test-extent"
  filesize = 10485760
}
`, pool)
}

func testAccISCSIExtentResourceConfigUpdated(pool string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_iscsi_extent" "test" {
  name     = "tf-acc-test-extent"
  type     = "FILE"
  path     = "/mnt/%s/iscsi-test-extent"
  filesize = 10485760
  comment  = "updated extent"
}
`, pool)
}
