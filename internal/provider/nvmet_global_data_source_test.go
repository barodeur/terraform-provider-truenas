package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNVMeTGlobalDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNVMeTGlobalDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_nvmet_global.test", "id"),
					resource.TestCheckResourceAttrSet("data.truenas_nvmet_global.test", "basenqn"),
				),
			},
		},
	})
}

func testAccNVMeTGlobalDataSourceConfig() string {
	return testAccProviderConfig() + `
data "truenas_nvmet_global" "test" {
}
`
}
