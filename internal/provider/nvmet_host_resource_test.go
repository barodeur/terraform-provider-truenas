package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNVMeTHostResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTHostResourceConfig("nqn.2014-08.org.nvmexpress:uuid:a1b2c3d4-0001-0002-0003-a1b2c3d4e5f6"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_host.test", "id"),
					resource.TestCheckResourceAttr("truenas_nvmet_host.test", "hostnqn", "nqn.2014-08.org.nvmexpress:uuid:a1b2c3d4-0001-0002-0003-a1b2c3d4e5f6"),
					resource.TestCheckResourceAttr("truenas_nvmet_host.test", "dhchap_hash", "SHA-256"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_host.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNVMeTHostResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTHostResourceConfig("nqn.2014-08.org.nvmexpress:uuid:b2c3d4e5-0002-0003-0004-b2c3d4e5f6a7"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_host.test", "hostnqn", "nqn.2014-08.org.nvmexpress:uuid:b2c3d4e5-0002-0003-0004-b2c3d4e5f6a7"),
				),
			},
			{
				Config: testAccNVMeTHostResourceConfigWithHash("nqn.2014-08.org.nvmexpress:uuid:b2c3d4e5-0002-0003-0004-b2c3d4e5f6a7", "SHA-512"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_host.test", "dhchap_hash", "SHA-512"),
				),
			},
		},
	})
}

func testAccNVMeTHostResourceConfig(hostnqn string) string {
	return testAccProviderConfig() + `
resource "truenas_nvmet_host" "test" {
  hostnqn = "` + hostnqn + `"
}
`
}

func testAccNVMeTHostResourceConfigWithHash(hostnqn, hash string) string {
	return testAccProviderConfig() + `
resource "truenas_nvmet_host" "test" {
  hostnqn     = "` + hostnqn + `"
  dhchap_hash = "` + hash + `"
}
`
}
