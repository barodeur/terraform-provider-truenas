package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccPoolName() string {
	if v := os.Getenv("TRUENAS_POOL"); v != "" {
		return v
	}
	return "tank"
}

func TestAccPoolDatasetResource_basic(t *testing.T) {
	pool := testAccPoolName()
	dsName := pool + "/tf-acc-test-basic"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDatasetResourceConfig(dsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "name", dsName),
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "id", dsName),
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "pool", pool),
					resource.TestCheckResourceAttrSet("truenas_pool_dataset.test", "mountpoint"),
				),
			},
			{
				ResourceName:            "truenas_pool_dataset.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"create_ancestors"},
			},
		},
	})
}

func TestAccPoolDatasetResource_fullOptions(t *testing.T) {
	pool := testAccPoolName()
	dsName := pool + "/tf-acc-test-full"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDatasetResourceConfigFull(dsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "name", dsName),
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "compression", "LZ4"),
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "atime", "OFF"),
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "sync", "STANDARD"),
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "copies", "2"),
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "comments", "managed by terraform"),
				),
			},
		},
	})
}

func TestAccPoolDatasetResource_update(t *testing.T) {
	pool := testAccPoolName()
	dsName := pool + "/tf-acc-test-update"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDatasetResourceConfigWithCompression(dsName, "LZ4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "compression", "LZ4"),
				),
			},
			{
				Config: testAccPoolDatasetResourceConfigWithCompression(dsName, "ZSTD"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "compression", "ZSTD"),
				),
			},
		},
	})
}

func TestAccPoolDatasetResource_nested(t *testing.T) {
	pool := testAccPoolName()
	dsName := pool + "/tf-acc-test-nested/child"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDatasetResourceConfigNested(dsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "name", dsName),
					resource.TestCheckResourceAttr("truenas_pool_dataset.test", "pool", pool),
				),
			},
		},
	})
}

func testAccPoolDatasetResourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_pool_dataset" "test" {
  name = %q
}
`, name)
}

func testAccPoolDatasetResourceConfigFull(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_pool_dataset" "test" {
  name        = %q
  compression = "LZ4"
  atime       = "OFF"
  sync        = "STANDARD"
  copies      = 2
  comments    = "managed by terraform"
}
`, name)
}

func testAccPoolDatasetResourceConfigWithCompression(name, compression string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_pool_dataset" "test" {
  name        = %q
  compression = %q
}
`, name, compression)
}

func testAccPoolDatasetResourceConfigNested(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_pool_dataset" "test" {
  name             = %q
  create_ancestors = true
}
`, name)
}
