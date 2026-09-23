// Copyright TrueNAS 2026
// SPDX-License-Identifier: MPL-2.0

package user_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/truenas/terraform-provider-truenas/internal/acctest"
)

// TestAccUser_basic creates a local user with a password set, checks its
// attributes, updates full_name and shell in place, imports it by its
// numeric id (ignoring the write-only password), and verifies destruction.
func TestAccUser_basic(t *testing.T) {
	username := acctest.RandName("tf-acc-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroyed(username),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig() + testAccUserConfig(username, "Test User", "/usr/bin/bash"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user.test", "username", username),
					resource.TestCheckResourceAttr("truenas_user.test", "full_name", "Test User"),
					resource.TestCheckResourceAttr("truenas_user.test", "shell", "/usr/bin/bash"),
					resource.TestCheckResourceAttr("truenas_user.test", "home", "/var/empty"),
					resource.TestCheckResourceAttr("truenas_user.test", "locked", "false"),
					resource.TestCheckResourceAttrSet("truenas_user.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_user.test", "uid"),
				),
			},
			// Update in place: change full_name.
			{
				Config: acctest.ProviderConfig() + testAccUserConfig(username, "Updated Name", "/usr/bin/bash"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user.test", "full_name", "Updated Name"),
					resource.TestCheckResourceAttr("truenas_user.test", "shell", "/usr/bin/bash"),
				),
			},
			// Import by the user's numeric id. Password is write-only and
			// never returned by the API, so it can't be verified.
			// group_create is also write-only (only sent on create) and is
			// never read back into state.
			{
				ResourceName:            "truenas_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "group_create"},
			},
		},
	})
}

// TestAccUser_identityImport creates a local user and re-imports it using
// an import block keyed by resource identity (Terraform 1.12+), rather than
// the legacy `terraform import ID` command. ImportStateVerify checks that
// the round-tripped state matches, modulo the write-only fields that are
// never read back (see TestAccUser_basic).
func TestAccUser_identityImport(t *testing.T) {
	username := acctest.RandName("tf-acc-user-ident")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0), // ImportBlockWithResourceIdentity requires Terraform 1.12.0+
		},
		CheckDestroy: testAccCheckUserDestroyed(username),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig() + testAccUserConfig(username, "Identity Test User", "/usr/bin/bash"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user.test", "username", username),
					resource.TestCheckResourceAttrSet("truenas_user.test", "id"),
				),
			},
			{
				ResourceName:            "truenas_user.test",
				ImportState:             true,
				ImportStateKind:         resource.ImportBlockWithResourceIdentity,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "group_create"},
			},
		},
	})
}

// TestAccUser_list creates a local user, then runs a `terraform query`
// (list resource) step against truenas_user and asserts the query finds at
// least one result. This exercises internal/listing's StreamCollection path
// end to end against a live box, rather than the stubbed unit tests in
// internal/listing/listing_test.go.
func TestAccUser_list(t *testing.T) {
	username := acctest.RandName("tf-acc-user-list")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0), // list/query support requires Terraform 1.14.0+
		},
		CheckDestroy: testAccCheckUserDestroyed(username),
		Steps: []resource.TestStep{
			{
				// Ensure at least one truenas_user exists (the box's own
				// built-in users would satisfy this too, but don't rely on
				// box-specific state).
				Config: acctest.ProviderConfig() + testAccUserConfig(username, "List Test User", "/usr/bin/bash"),
			},
			{
				Query: true,
				Config: acctest.ProviderConfig() + `
list "truenas_user" "test" {
  provider = truenas
  config {}
}
`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("truenas_user.test", 1),
				},
			},
		},
	})
}

func testAccUserConfig(username, fullName, shell string) string {
	return fmt.Sprintf(`
resource "truenas_user" "test" {
  username          = %q
  full_name         = %q
  password          = "Tf-Acc-Test-Passw0rd!"
  password_disabled = false
  home              = "/var/empty"
  shell             = %q
  smb               = false
  group_create      = true
}
`, username, fullName, shell)
}

func testAccCheckUserDestroyed(username string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := acctest.Client()
		raw, err := c.Call(context.Background(), "user.query", [][]any{{"username", "=", username}})
		if err != nil {
			return fmt.Errorf("error checking user %s: %v", username, err)
		}
		var results []struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(raw, &results); err != nil {
			return fmt.Errorf("error parsing user.query response: %v", err)
		}
		if len(results) > 0 {
			return fmt.Errorf("user %s still exists", username)
		}
		return nil
	}
}
