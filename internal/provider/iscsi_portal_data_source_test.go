package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSIPortalDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIPortalDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_iscsi_portal.test", "id"),
					resource.TestCheckResourceAttrSet("data.truenas_iscsi_portal.test", "tag"),
					resource.TestCheckResourceAttr("data.truenas_iscsi_portal.test", "listen.#", "1"),
				),
			},
		},
	})
}

func testAccISCSIPortalDataSourceConfig() string {
	return testAccProviderConfig() + `
resource "truenas_iscsi_portal" "test" {
  listen {
    ip = "0.0.0.0"
  }
}

data "truenas_iscsi_portal" "test" {
  id = truenas_iscsi_portal.test.id
}
`
}
