package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccCheckCronjobDestroy(s *terraform.State) error {
	ctx := context.Background()
	host := os.Getenv("TRUENAS_HOST")
	apiKey := os.Getenv("TRUENAS_API_KEY")
	c, err := client.NewClient(ctx, host, apiKey, true)
	if err != nil {
		return fmt.Errorf("creating client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "truenas_cronjob" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("parsing ID %q: %s", rs.Primary.ID, err)
		}

		var results []cronjobResult
		err = c.Call(ctx, "cronjob.query", []any{
			[]any{[]any{"id", "=", id}},
		}, &results)
		if err != nil {
			return fmt.Errorf("querying cronjob: %s", err)
		}
		if len(results) > 0 {
			return fmt.Errorf("cronjob %d still exists", id)
		}
	}

	return nil
}

func TestAccCronjobResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCronjobDestroy,
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
					resource.TestCheckResourceAttr("truenas_cronjob.test", "stdout", "true"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "stderr", "false"),
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
		CheckDestroy:             testAccCheckCronjobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCronjobResourceConfig("echo before", "root", "0", "1", "*", "*", "*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cronjob.test", "command", "echo before"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.minute", "0"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.hour", "1"),
				),
			},
			{
				Config: testAccCronjobResourceConfig("echo after", "root", "30", "2", "*", "*", "*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cronjob.test", "command", "echo after"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.minute", "30"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.hour", "2"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.dom", "*"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.month", "*"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.dow", "*"),
				),
			},
		},
	})
}

func TestAccCronjobResource_allFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCronjobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCronjobResourceConfigFull("echo full", "root", "Backup job", true, false, true, "15", "3", "1", "6", "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cronjob.test", "command", "echo full"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "user", "root"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "description", "Backup job"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "enabled", "true"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "stdout", "false"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "stderr", "true"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.minute", "15"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.hour", "3"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.dom", "1"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.month", "6"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.dow", "0"),
				),
			},
			{
				Config: testAccCronjobResourceConfigFull("echo updated", "root", "Updated job", false, true, false, "45", "12", "*", "*", "5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cronjob.test", "command", "echo updated"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "description", "Updated job"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "enabled", "false"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "stdout", "true"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "stderr", "false"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.minute", "45"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.hour", "12"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "schedule.dow", "5"),
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

func TestAccCronjobResource_defaults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCronjobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCronjobResourceConfig("echo defaults", "root", "0", "0", "*", "*", "*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cronjob.test", "enabled", "true"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "stdout", "true"),
					resource.TestCheckResourceAttr("truenas_cronjob.test", "stderr", "false"),
					resource.TestCheckNoResourceAttr("truenas_cronjob.test", "description"),
				),
			},
		},
	})
}

func TestAccCronjobResource_removeDescription(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCronjobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCronjobResourceConfigFull("echo desc", "root", "Has description", true, true, false, "0", "0", "*", "*", "*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_cronjob.test", "description", "Has description"),
				),
			},
			{
				Config: testAccCronjobResourceConfig("echo desc", "root", "0", "0", "*", "*", "*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("truenas_cronjob.test", "description"),
				),
			},
		},
	})
}

func testAccCronjobResourceConfig(command, user, minute, hour, dom, month, dow string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_cronjob" "test" {
  command = %[1]q
  user    = %[2]q
  schedule = {
    minute = %[3]q
    hour   = %[4]q
    dom    = %[5]q
    month  = %[6]q
    dow    = %[7]q
  }
}
`, command, user, minute, hour, dom, month, dow)
}

func testAccCronjobResourceConfigFull(command, user, description string, enabled, stdout, stderr bool, minute, hour, dom, month, dow string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_cronjob" "test" {
  command     = %[1]q
  user        = %[2]q
  description = %[3]q
  enabled     = %[4]t
  stdout      = %[5]t
  stderr      = %[6]t
  schedule = {
    minute = %[7]q
    hour   = %[8]q
    dom    = %[9]q
    month  = %[10]q
    dow    = %[11]q
  }
}
`, command, user, description, enabled, stdout, stderr, minute, hour, dom, month, dow)
}
