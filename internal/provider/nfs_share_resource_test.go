package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNFSShareResource_basic(t *testing.T) {
	pool := testAccPoolName()
	dsName := pool + "/tf-acc-test-nfs"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNFSShareResourceConfig(dsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nfs_share.test", "id"),
					resource.TestCheckResourceAttr("truenas_nfs_share.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "truenas_nfs_share.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNFSShareResource_withOptions(t *testing.T) {
	pool := testAccPoolName()
	dsName := pool + "/tf-acc-test-nfs-opts"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNFSShareResourceConfigWithOptions(dsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nfs_share.test", "comment", "test nfs share"),
					resource.TestCheckResourceAttr("truenas_nfs_share.test", "maproot_user", "root"),
				),
			},
			{
				Config: testAccNFSShareResourceConfigWithOptionsUpdate(dsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nfs_share.test", "comment", "updated nfs share"),
				),
			},
		},
	})
}

func testAccNFSShareResourceConfig(dsName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_pool_dataset" "test_nfs" {
  name = %q
}

resource "truenas_nfs_share" "test" {
  path = truenas_pool_dataset.test_nfs.mountpoint
}
`, dsName)
}

func testAccNFSShareResourceConfigWithOptions(dsName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_pool_dataset" "test_nfs" {
  name = %q
}

resource "truenas_nfs_share" "test" {
  path         = truenas_pool_dataset.test_nfs.mountpoint
  comment      = "test nfs share"
  maproot_user = "root"
}
`, dsName)
}

func testAccNFSShareResourceConfigWithOptionsUpdate(dsName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_pool_dataset" "test_nfs" {
  name = %q
}

resource "truenas_nfs_share" "test" {
  path    = truenas_pool_dataset.test_nfs.mountpoint
  comment = "updated nfs share"
}
`, dsName)
}
