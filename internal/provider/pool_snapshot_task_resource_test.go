package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/barodeur/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccCheckPoolSnapshotTaskDestroy(s *terraform.State) error {
	ctx := context.Background()
	host := os.Getenv("TRUENAS_HOST")
	if !strings.HasPrefix(host, "ws://") && !strings.HasPrefix(host, "wss://") {
		host = "wss://" + host
	}
	apiKey := os.Getenv("TRUENAS_API_KEY")
	c, err := client.NewClient(ctx, host, apiKey, true)
	if err != nil {
		return fmt.Errorf("creating client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "truenas_pool_snapshot_task" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("parsing ID %q: %s", rs.Primary.ID, err)
		}

		var result poolSnapshotTaskResult
		err = c.Call(ctx, "pool.snapshottask.get_instance", []any{id}, &result)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
				continue
			}
			return fmt.Errorf("querying snapshot task: %s", err)
		}
		return fmt.Errorf("snapshot task %d still exists", id)
	}

	return nil
}

func TestAccPoolSnapshotTaskResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPoolSnapshotTaskDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolSnapshotTaskResourceConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_pool_snapshot_task.test", "id"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "dataset", "tank/snap-test"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "recursive", "false"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "lifetime_value", "2"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "lifetime_unit", "WEEK"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "enabled", "true"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "naming_schema", "auto-%Y-%m-%d_%H-%M"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "allow_empty", "true"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.minute", "00"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.hour", "*"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.dom", "*"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.month", "*"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.dow", "*"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.begin", "00:00"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.end", "23:59"),
				),
			},
			{
				ResourceName:      "truenas_pool_snapshot_task.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPoolSnapshotTaskResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPoolSnapshotTaskDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolSnapshotTaskResourceConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "lifetime_value", "2"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "lifetime_unit", "WEEK"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "enabled", "true"),
				),
			},
			{
				Config: testAccPoolSnapshotTaskResourceConfig_updated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "dataset", "tank/snap-test"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "lifetime_value", "30"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "lifetime_unit", "DAY"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "enabled", "false"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.minute", "30"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.hour", "2"),
				),
			},
		},
	})
}

func TestAccPoolSnapshotTaskResource_allFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPoolSnapshotTaskDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolSnapshotTaskResourceConfig_allFields(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_pool_snapshot_task.test", "id"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "dataset", "tank/snap-test"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "recursive", "true"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "lifetime_value", "7"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "lifetime_unit", "DAY"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "enabled", "true"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "naming_schema", "daily-%Y-%m-%d_%H-%M"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "allow_empty", "false"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "exclude.#", "1"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "exclude.0", "tank/snap-test/child"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.minute", "15"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.hour", "3"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.dom", "1"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.month", "*"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.dow", "*"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.begin", "01:00"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.end", "05:00"),
				),
			},
			{
				ResourceName:      "truenas_pool_snapshot_task.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPoolSnapshotTaskResource_partialSchedule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPoolSnapshotTaskDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolSnapshotTaskResourceConfig_partialSchedule(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.minute", "30"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.hour", "6"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.dom", "*"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.month", "*"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.dow", "*"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.begin", "00:00"),
					resource.TestCheckResourceAttr("truenas_pool_snapshot_task.test", "schedule.end", "23:59"),
				),
			},
		},
	})
}

func testAccPoolSnapshotTaskResourceConfig_basic() string {
	return testAccProviderConfig() + `
resource "truenas_pool_dataset" "snap_test" {
  name = "tank/snap-test"
}

resource "truenas_pool_snapshot_task" "test" {
  dataset = truenas_pool_dataset.snap_test.name
}
`
}

func testAccPoolSnapshotTaskResourceConfig_updated() string {
	return testAccProviderConfig() + `
resource "truenas_pool_dataset" "snap_test" {
  name = "tank/snap-test"
}

resource "truenas_pool_snapshot_task" "test" {
  dataset        = truenas_pool_dataset.snap_test.name
  lifetime_value = 30
  lifetime_unit  = "DAY"
  enabled        = false
  schedule = {
    minute = "30"
    hour   = "2"
    dom    = "*"
    month  = "*"
    dow    = "*"
    begin  = "00:00"
    end    = "23:59"
  }
}
`
}

func testAccPoolSnapshotTaskResourceConfig_allFields() string {
	return testAccProviderConfig() + `
resource "truenas_pool_dataset" "snap_test" {
  name = "tank/snap-test"
}

resource "truenas_pool_dataset" "snap_test_child" {
  name = "tank/snap-test/child"
}

resource "truenas_pool_snapshot_task" "test" {
  dataset        = truenas_pool_dataset.snap_test.name
  recursive      = true
  lifetime_value = 7
  lifetime_unit  = "DAY"
  enabled        = true
  exclude        = [truenas_pool_dataset.snap_test_child.name]
  naming_schema  = "daily-%Y-%m-%d_%H-%M"
  allow_empty    = false
  schedule = {
    minute = "15"
    hour   = "3"
    dom    = "1"
    month  = "*"
    dow    = "*"
    begin  = "01:00"
    end    = "05:00"
  }
}
`
}

func testAccPoolSnapshotTaskResourceConfig_partialSchedule() string {
	return testAccProviderConfig() + `
resource "truenas_pool_dataset" "snap_test" {
  name = "tank/snap-test"
}

resource "truenas_pool_snapshot_task" "test" {
  dataset = truenas_pool_dataset.snap_test.name
  schedule = {
    minute = "30"
    hour   = "6"
  }
}
`
}
