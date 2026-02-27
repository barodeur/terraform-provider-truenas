package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNVMeTHostSubsysResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTHostSubsysResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_host_subsys.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_host_subsys.test", "host_id"),
					resource.TestCheckResourceAttrSet("truenas_nvmet_host_subsys.test", "subsys_id"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_host_subsys.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccNVMeTHostSubsysResourceConfig() string {
	return testAccProviderConfig() + `
resource "truenas_nvmet_host" "hs_test" {
  hostnqn = "nqn.2014-08.org.nvmexpress:uuid:c3d4e5f6-0003-0004-0005-c3d4e5f6a7b8"
}

resource "truenas_nvmet_subsys" "hs_test" {
  name = "tf-acc-test-hs-subsys"
}

resource "truenas_nvmet_host_subsys" "test" {
  host_id   = truenas_nvmet_host.hs_test.id
  subsys_id = truenas_nvmet_subsys.hs_test.id
}
`
}
