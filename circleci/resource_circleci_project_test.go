package circleci

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccCircleCIFollowThenUnfollowProject(t *testing.T) {
	organization := "ZymoticB"
	project := "cci-tfp-test-repo"
	resourceName := "circleci_project.p1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOrgProviders,
		CheckDestroy: testAccCircleCIProjectCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCircleCIProject(organization, project),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", fmt.Sprintf("%s.%s", organization, project)),
					resource.TestCheckResourceAttr(resourceName, "project", project),
				),
			},
		},
	})
}

func testAccCircleCIProjectCheckDestroy(s *terraform.State) error {
	ctx := testAccOrgProvider.Meta().(ProviderContext)
	return projectCheckDestroy(ctx, s)
}

func projectCheckDestroy(ctx ProviderContext, s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "circleci_project" {
			continue
		}

		organization := rs.Primary.Attributes["organization"]
		if organization == "" {
			organization = ctx.Org
		}

		p, err := ctx.Client.GetProject(organization, rs.Primary.Attributes["project"])
		if err != nil {
			return err
		}

		if p != nil {
			return errors.New("Project should have been unfollowed")
		}
	}

	return nil
}

func testAccCircleCIProject(org, project string) string {
	return fmt.Sprintf(`
resource "circleci_project" "p1" {
  organization = "%s"
  project = "%s"
}
`, org, project)
}
