package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCronjobDataSource_byID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCronjobDestroy,
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
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "enabled", "true"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "stdout", "true"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "stderr", "false"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "schedule.minute", "0"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "schedule.hour", "0"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "schedule.dom", "*"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "schedule.month", "*"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "schedule.dow", "*"),
				),
			},
		},
	})
}

func TestAccCronjobDataSource_allFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCronjobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCronjobDataSourceAllFieldsConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.truenas_cronjob.test", "id",
						"truenas_cronjob.test", "id",
					),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "command", "echo ds-full"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "user", "root"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "description", "DS full test"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "enabled", "false"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "stdout", "false"),
					resource.TestCheckResourceAttr("data.truenas_cronjob.test", "stderr", "true"),
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

func testAccCronjobDataSourceAllFieldsConfig() string {
	return testAccProviderConfig() + `
resource "truenas_cronjob" "test" {
  command     = "echo ds-full"
  user        = "root"
  description = "DS full test"
  enabled     = false
  stdout      = false
  stderr      = true
  schedule = {
    minute = "30"
    hour   = "6"
    dom    = "*"
    month  = "*"
    dow    = "1"
  }
}

data "truenas_cronjob" "test" {
  id = truenas_cronjob.test.id
}
`
}
