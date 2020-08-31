package circleci

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceCircleCIProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceCircleCIProjectCreate,
		Read:   resourceCircleCIProjectRead,
		Delete: resourceCircleCIProjectDelete,
		Exists: resourceCircleCIProjectExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"organization": {
				Description: "The CircleCI organization.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			"project": {
				Description: "The name of the CircleCI project to create the variable in",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
		},
		SchemaVersion: 1,
	}
}

func resourceCircleCIProjectCreate(d *schema.ResourceData, meta interface{}) error {
	providerContext := meta.(ProviderContext)
	providerClient := providerContext.Client

	projectName := d.Get("project").(string)

	organization := getOrganization(d, providerContext)
	if organization == "" {
		return errors.New("organization must be set at the project, or provider level")
	}

	p, err := providerClient.GetProject(organization, projectName)
	if err != nil {
		return err
	}

	if p != nil {
		return fmt.Errorf("%s/%s on %s is already followed", organization, projectName, providerContext.VCS)
	}

	if _, err := providerClient.FollowProject(providerContext.VCS, organization, projectName); err != nil {
		return err
	}

	d.SetId(generateProjectId(organization, projectName))
	return resourceCircleCIProjectRead(d, meta)
}

func resourceCircleCIProjectRead(d *schema.ResourceData, meta interface{}) error {
	providerContext := meta.(ProviderContext)
	providerClient := providerContext.Client

	// If we don't have a project name we're doing an import. Parse it from the ID.
	if _, ok := d.GetOk("project"); !ok {
		if err := setDataFromId(d); err != nil {
			return err
		}
	}

	projectName := d.Get("project").(string)
	organization := getOrganization(d, providerContext)
	if organization == "" {
		return errors.New("organization must be set at the project, or provider level")
	}

	p, err := providerClient.GetProject(organization, projectName)
	if err != nil {
		return err
	}

	if p == nil {
		return fmt.Errorf("%s/%s is not found in %s", organization, projectName, providerContext.VCS)
	}

	if err := d.Set("project", p.Reponame); err != nil {
		return err
	}

	return nil
}

func resourceCircleCIProjectDelete(d *schema.ResourceData, meta interface{}) error {
	providerContext := meta.(ProviderContext)
	providerClient := providerContext.Client

	projectName := d.Get("project").(string)

	organization := getOrganization(d, providerContext)
	if organization == "" {
		return errors.New("organization must be set at the project, or provider level")
	}

	_, err := providerClient.UnfollowProject(providerContext.VCS, organization, projectName)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func resourceCircleCIProjectExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	providerContext := meta.(ProviderContext)
	providerClient := providerContext.Client

	// If we don't have a project name we're doing an import. Parse it from the ID.
	if _, ok := d.GetOk("project"); !ok {
		if err := setDataFromId(d); err != nil {
			return false, err
		}
	}

	projectName := d.Get("project").(string)

	organization := getOrganization(d, providerContext)
	if organization == "" {
		return false, errors.New("organization must be set at the project, or provider level")
	}

	p, err := providerClient.GetProject(organization, projectName)
	if err != nil {
		return false, err
	}

	return p != nil, nil
}

func setDataFromId(d *schema.ResourceData) error {
	parts := strings.Split(d.Id(), ".")

	if len(parts) != 2 {
		return fmt.Errorf("expected project ID to be of the format {org}.{repo}. Please rename to eanble import")
	}

	_ = d.Set("organization", parts[0])
	_ = d.Set("project", parts[1])
	return nil
}

func generateProjectId(organization, projectName string) string {
	vars := []string{
		organization,
		projectName,
	}
	return strings.Join(vars, ".")
}
