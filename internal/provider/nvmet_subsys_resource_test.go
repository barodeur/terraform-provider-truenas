package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNVMeTSubsysResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTSubsysResourceConfig("tf-acc-test-subsys"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_subsys.test", "id"),
					resource.TestCheckResourceAttr("truenas_nvmet_subsys.test", "name", "tf-acc-test-subsys"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_subsys.test", "subnqn"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_subsys.test", "serial"),
					resource.TestCheckResourceAttr("truenas_nvmet_subsys.test", "allow_any_host", "false"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_subsys.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNVMeTSubsysResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTSubsysResourceConfig("tf-acc-test-subsys-upd"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_subsys.test", "allow_any_host", "false"),
				),
			},
			{
				Config: testAccNVMeTSubsysResourceConfigAllowAny("tf-acc-test-subsys-upd", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_subsys.test", "allow_any_host", "true"),
				),
			},
		},
	})
}

func testAccNVMeTSubsysResourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_nvmet_subsys" "test" {
  name = %q
}
`, name)
}

func testAccNVMeTSubsysResourceConfigAllowAny(name string, allowAny bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_nvmet_subsys" "test" {
  name           = %q
  allow_any_host = %t
}
`, name, allowAny)
}
