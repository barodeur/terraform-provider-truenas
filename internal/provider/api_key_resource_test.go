package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAPIKeyResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig("tf-acc-test-basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_api_key.test", "name", "tf-acc-test-basic"),
					resource.TestCheckResourceAttrSet("truenas_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_api_key.test", "key"),
					resource.TestCheckResourceAttrSet("truenas_api_key.test", "created_at"),
					resource.TestCheckResourceAttrSet("truenas_api_key.test", "username"),
					resource.TestCheckResourceAttr("truenas_api_key.test", "revoked", "false"),
				),
			},
			{
				ResourceName:            "truenas_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
		},
	})
}

func TestAccAPIKeyResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig("tf-acc-test-before"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_api_key.test", "name", "tf-acc-test-before"),
				),
			},
			{
				Config: testAccAPIKeyResourceConfig("tf-acc-test-after"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_api_key.test", "name", "tf-acc-test-after"),
				),
			},
		},
	})
}

func testAccAPIKeyResourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_api_key" "test" {
  name = %q
}
`, name)
}
