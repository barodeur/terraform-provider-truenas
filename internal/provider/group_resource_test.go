package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupResourceConfig("tf-acc-test-group"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_group.test", "name", "tf-acc-test-group"),
					resource.TestCheckResourceAttrSet("truenas_group.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_group.test", "gid"),
					resource.TestCheckResourceAttr("truenas_group.test", "builtin", "false"),
				),
			},
			{
				ResourceName:            "truenas_group.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"allow_duplicate_gid"},
			},
		},
	})
}

func TestAccGroupResource_withSmb(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupResourceConfigWithSmb("tf-acc-test-group-smb", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_group.test", "name", "tf-acc-test-group-smb"),
					resource.TestCheckResourceAttr("truenas_group.test", "smb", "true"),
				),
			},
			{
				Config: testAccGroupResourceConfigWithSmb("tf-acc-test-group-smb", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_group.test", "smb", "false"),
				),
			},
		},
	})
}

func testAccGroupResourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_group" "test" {
  name = %q
}
`, name)
}

func testAccGroupResourceConfigWithSmb(name string, smb bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_group" "test" {
  name = %q
  smb  = %t
}
`, name, smb)
}
