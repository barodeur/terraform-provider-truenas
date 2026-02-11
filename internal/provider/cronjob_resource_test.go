package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCronjobResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCronjobResourceConfig("echo hello", "root", "00", "*/2", "*", "*", "*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cronjob.test", "command", "echo hello"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "user", "root"),
					resource.TestCheckResourceAttrSet("truenas_cronjob.test", "id"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.minute", "00"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.hour", "*/2"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.dom", "*"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.month", "*"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.dow", "*"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "truenas_cronjob.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccCronjobResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCronjobResourceConfig("echo before", "root", "0", "1", "*", "*", "*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cronjob.test", "command", "echo before"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.hour", "1"),
				),
			},
			{
				Config: testAccCronjobResourceConfig("echo after", "root", "30", "2", "*", "*", "*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cronjob.test", "command", "echo after"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.minute", "30"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.hour", "2"),
				),
			},
		},
	})
}

func testAccCronjobResourceConfig(command, user, minute, hour, dom, month, dow string) string {
	return testAccProviderConfig() + `
resource "truenas_cronjob" "test" {
  command = "` + command + `"
  user    = "` + user + `"
  schedule = {
    minute = "` + minute + `"
    hour   = "` + hour + `"
    dom    = "` + dom + `"
    month  = "` + month + `"
    dow    = "` + dow + `"
  }
}
`
}
