package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccISCSIGlobalDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccISCSIGlobalDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_iscsi_global.test", "id"),
					resource.TestCheckResourceAttrSet("data.truenas_iscsi_global.test", "basename"),
					resource.TestCheckResourceAttrSet("data.truenas_iscsi_global.test", "listen_port"),
				),
			},
		},
	})
}

func testAccISCSIGlobalDataSourceConfig() string {
	return testAccProviderConfig() + `
data "truenas_iscsi_global" "test" {}
`
}
