package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSMBShareResource_basic(t *testing.T) {
	pool := testAccPoolName()
	dsName := pool + "/tf-acc-test-smb"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSMBShareResourceConfig(dsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_smb_share.test", "id"),
					resource.TestCheckResourceAttr("truenas_smb_share.test", "name", "tf-acc-test-smb"),
					resource.TestCheckResourceAttr("truenas_smb_share.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "truenas_smb_share.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSMBShareResource_update(t *testing.T) {
	pool := testAccPoolName()
	dsName := pool + "/tf-acc-test-smb-update"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSMBShareResourceConfigWithComment(dsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_smb_share.test", "comment", "initial comment"),
				),
			},
			{
				Config: testAccSMBShareResourceConfigUpdated(dsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_smb_share.test", "readonly", "true"),
				),
			},
		},
	})
}

func testAccSMBShareResourceConfig(dsName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_pool_dataset" "test_smb" {
  name = %q
}

resource "truenas_smb_share" "test" {
  name = "tf-acc-test-smb"
  path = truenas_pool_dataset.test_smb.mountpoint
}
`, dsName)
}

func testAccSMBShareResourceConfigWithComment(dsName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_pool_dataset" "test_smb" {
  name = %q
}

resource "truenas_smb_share" "test" {
  name    = "tf-acc-test-smb-update"
  path    = truenas_pool_dataset.test_smb.mountpoint
  comment = "initial comment"
}
`, dsName)
}

func testAccSMBShareResourceConfigUpdated(dsName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_pool_dataset" "test_smb" {
  name = %q
}

resource "truenas_smb_share" "test" {
  name     = "tf-acc-test-smb-update"
  path     = truenas_pool_dataset.test_smb.mountpoint
  readonly = true
}
`, dsName)
}
