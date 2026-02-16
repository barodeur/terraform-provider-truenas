package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSITargetextentResource_basic(t *testing.T) {
	pool := testAccPoolName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSITargetextentResourceConfig(pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_iscsi_targetextent.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_iscsi_targetextent.test", "lunid"),
					resource.TestCheckResourceAttrPair(
						"truenas_iscsi_targetextent.test", "target",
						"truenas_iscsi_target.test_te", "id",
					),
					resource.TestCheckResourceAttrPair(
						"truenas_iscsi_targetextent.test", "extent",
						"truenas_iscsi_extent.test_te", "id",
					),
				),
			},
			{
				ResourceName:      "truenas_iscsi_targetextent.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccISCSITargetextentResourceConfig(pool string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_iscsi_portal" "test_te" {
  listen {
    ip = "0.0.0.0"
  }
}

resource "truenas_iscsi_target" "test_te" {
  name = "tf-acc-test-te-target"
  groups {
    portal = truenas_iscsi_portal.test_te.id
  }
}

resource "truenas_iscsi_extent" "test_te" {
  name     = "tf-acc-test-te-extent"
  type     = "FILE"
  path     = "/mnt/%s/iscsi-test-te-extent"
  filesize = 10485760
}

resource "truenas_iscsi_targetextent" "test" {
  target = truenas_iscsi_target.test_te.id
  extent = truenas_iscsi_extent.test_te.id
}
`, pool)
}
