package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig("tfaccuser", "TF Acc Test User"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user.test", "username", "tfaccuser"),
					resource.TestCheckResourceAttr("truenas_user.test", "full_name", "TF Acc Test User"),
					resource.TestCheckResourceAttrSet("truenas_user.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_user.test", "uid"),
					resource.TestCheckResourceAttrSet("truenas_user.test", "group"),
					resource.TestCheckResourceAttr("truenas_user.test", "builtin", "false"),
				),
			},
			{
				ResourceName:            "truenas_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "group_create", "home_create"},
			},
		},
	})
}

func TestAccUserResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig("tfaccuserupd", "Before Update"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user.test", "full_name", "Before Update"),
				),
			},
			{
				Config: testAccUserResourceConfig("tfaccuserupd", "After Update"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user.test", "full_name", "After Update"),
				),
			},
		},
	})
}

func TestAccUserResource_withGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfigWithGroup(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user.test", "username", "tfaccusergrp"),
					resource.TestCheckResourceAttr("truenas_user.test", "smb", "true"),
				),
			},
		},
	})
}

func testAccUserResourceConfig(username, fullName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "truenas_user" "test" {
  username          = %q
  full_name         = %q
  smb               = false
  password_disabled = true
}
`, username, fullName)
}

func testAccUserResourceConfigWithGroup() string {
	return testAccProviderConfig() + `
resource "truenas_group" "test" {
  name = "tf-acc-test-usergrp"
  smb  = true
}

resource "truenas_user" "test" {
  username  = "tfaccusergrp"
  full_name = "TF Acc User With Group"
  group     = truenas_group.test.id
  smb       = true
  password  = "TestPassword123!"
}
`
}
