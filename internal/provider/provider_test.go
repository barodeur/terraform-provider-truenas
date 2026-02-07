package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"truenas": providerserver.NewProtocol6WithError(New()()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TRUENAS_HOST") == "" {
		t.Fatal("TRUENAS_HOST must be set for acceptance tests")
	}
	if os.Getenv("TRUENAS_API_KEY") == "" {
		t.Fatal("TRUENAS_API_KEY must be set for acceptance tests")
	}
}

func testAccProviderConfig() string {
	return `
provider "truenas" {
  host     = "` + os.Getenv("TRUENAS_HOST") + `"
  api_key  = "` + os.Getenv("TRUENAS_API_KEY") + `"
  insecure = true
}
`
}
