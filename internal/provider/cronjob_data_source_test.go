package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCronjobDataSource_byID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCronjobDataSourceByIDConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.truenas_cronjob.test", "id",
						"truenas_cronjob.test", "id",
					),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "command", "echo ds-test"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "user", "root"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "schedule.minute", "0"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "schedule.hour", "0"),
				),
			},
		},
	})
}

func testAccCronjobDataSourceByIDConfig() string {
	return testAccProviderConfig() + `
resource "truenas_cronjob" "test" {
  command = "echo ds-test"
  user    = "root"
  schedule = {
    minute = "0"
    hour   = "0"
    dom    = "*"
    month  = "*"
    dow    = "*"
  }
}

data "truenas_cronjob" "test" {
  id = truenas_cronjob.test.id
}
`
}
