package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPoolDataSource_byName(t *testing.T) {
	pool := testAccPoolName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDataSourceConfigByName(pool),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.truenas_pool.test", "name", pool),
					resource.TestCheckResourceAttrSet("data.truenas_pool.test", "id"),
					resource.TestCheckResourceAttrSet("data.truenas_pool.test", "path"),
					resource.TestCheckResourceAttrSet("data.truenas_pool.test", "status"),
					resource.TestCheckResourceAttrSet("data.truenas_pool.test", "healthy"),
				),
			},
		},
	})
}

func testAccPoolDataSourceConfigByName(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
data "truenas_pool" "test" {
  name = %q
}
`, name)
}
