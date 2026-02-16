package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivilegeResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivilegeResourceConfig("tf-acc-test-priv", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_privilege.test", "name", "tf-acc-test-priv"),
					resource.TestCheckResourceAttrSet("truenas_privilege.test", "id"),
					resource.TestCheckResourceAttr("truenas_privilege.test", "web_shell", "false"),
				),
			},
			{
				ResourceName:      "truenas_privilege.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPrivilegeResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivilegeResourceConfig("tf-acc-test-priv-upd", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_privilege.test", "web_shell", "false"),
				),
			},
			{
				Config: testAccPrivilegeResourceConfig("tf-acc-test-priv-upd", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_privilege.test", "web_shell", "true"),
				),
			},
		},
	})
}

func TestAccPrivilegeResource_withLocalGroups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivilegeResourceConfigWithLocalGroups(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_privilege.test", "name", "tf-acc-test-priv-groups"),
					resource.TestCheckResourceAttr("truenas_privilege.test", "roles.#", "1"),
					resource.TestCheckResourceAttr("truenas_privilege.test", "roles.0", "READONLY_ADMIN"),
					resource.TestCheckResourceAttr("truenas_privilege.test", "local_groups.#", "1"),
					resource.TestCheckResourceAttr("truenas_privilege.test", "web_shell", "false"),
				),
			},
		},
	})
}

func testAccPrivilegeResourceConfig(name string, webShell bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_privilege" "test" {
  name      = %q
  web_shell = %t
}
`, name, webShell)
}

func testAccPrivilegeResourceConfigWithLocalGroups() string {
	return testAccProviderConfig() + `
resource "truenas_group" "test" {
  name = "tf-acc-test-priv-grp"
}

resource "truenas_privilege" "test" {
  name         = "tf-acc-test-priv-groups"
  local_groups = [truenas_group.test.id]
  roles        = ["READONLY_ADMIN"]
  web_shell    = false
}
`
}
