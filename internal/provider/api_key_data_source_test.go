package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAPIKeyDataSource_byName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyDataSourceByNameConfig("tf-acc-test-ds-name"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.truenas_api_key.test", "name", "tf-acc-test-ds-name"),
					resource.TestCheckResourceAttrPair(
						"data.truenas_api_key.test", "id",
						"truenas_api_key.test", "id",
					),
					resource.TestCheckResourceAttrSet("data.truenas_api_key.test", "username"),
					resource.TestCheckResourceAttrSet("data.truenas_api_key.test", "created_at"),
					resource.TestCheckResourceAttr("data.truenas_api_key.test", "revoked", "false"),
				),
			},
		},
	})
}

func TestAccAPIKeyDataSource_byID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyDataSourceByIDConfig("tf-acc-test-ds-id"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.truenas_api_key.test", "name", "tf-acc-test-ds-id"),
					resource.TestCheckResourceAttrPair(
						"data.truenas_api_key.test", "id",
						"truenas_api_key.test", "id",
					),
				),
			},
		},
	})
}

func testAccAPIKeyDataSourceByNameConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_api_key" "test" {
  name = %[1]q
}

data "truenas_api_key" "test" {
  name = truenas_api_key.test.name
}
`, name)
}

func testAccAPIKeyDataSourceByIDConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_api_key" "test" {
  name = %[1]q
}

data "truenas_api_key" "test" {
  id = truenas_api_key.test.id
}
`, name)
}
