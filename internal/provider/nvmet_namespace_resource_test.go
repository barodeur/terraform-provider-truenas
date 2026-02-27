package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNVMeTNamespaceResource_basic(t *testing.T) {
	pool := testAccPoolName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTNamespaceResourceConfig(pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_namespace.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_namespace.test", "nsid"),
					resource.TestCheckResourceAttr("truenas_nvmet_namespace.test", "device_type", "FILE"),
					resource.TestCheckResourceAttr("truenas_nvmet_namespace.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_namespace.test", "device_uuid"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_namespace.test", "device_nguid"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_namespace.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNVMeTNamespaceResource_update(t *testing.T) {
	pool := testAccPoolName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTNamespaceResourceConfig(pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_namespace.test", "enabled", "true"),
				),
			},
			{
				Config: testAccNVMeTNamespaceResourceConfigDisabled(pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_namespace.test", "enabled", "false"),
				),
			},
		},
	})
}

func testAccNVMeTNamespaceResourceConfig(pool string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_nvmet_subsys" "ns_test" {
  name           = "tf-acc-test-ns-subsys"
  allow_any_host = true
}

resource "truenas_nvmet_namespace" "test" {
  subsys_id   = truenas_nvmet_subsys.ns_test.id
  device_type = "FILE"
  device_path = "/mnt/%s/nvmet-ns-test"
  filesize    = 1073741824
}
`, pool)
}

func testAccNVMeTNamespaceResourceConfigDisabled(pool string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_nvmet_subsys" "ns_test" {
  name           = "tf-acc-test-ns-subsys"
  allow_any_host = true
}

resource "truenas_nvmet_namespace" "test" {
  subsys_id   = truenas_nvmet_subsys.ns_test.id
  device_type = "FILE"
  device_path = "/mnt/%s/nvmet-ns-test"
  filesize    = 1073741824
  enabled     = false
}
`, pool)
}
