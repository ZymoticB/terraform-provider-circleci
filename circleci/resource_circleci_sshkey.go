package circleci

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"golang.org/x/crypto/ssh"
)

func resourceCircleCISSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceCircleCISSHKeyCreate,
		Read:   resourceCircleCISSHKeyRead,
		Delete: resourceCircleCISSHKeyDelete,
		Exists: resourceCircleCISSHKeyExists,
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
			"hostname": {
				Description: "The hostname for which to use this SSH Key for authentication",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"private_key": {
				Description: "The private key of the SSH Key in RSA format",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Sensitive:   true,
			},
			"fingerprint": {
				Description: "The fingerprint of the SSH private key",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
		SchemaVersion: 1,
	}
}

func resourceCircleCISSHKeyCreate(d *schema.ResourceData, meta interface{}) error {
	providerContext := meta.(ProviderContext)
	providerClient := providerContext.Client

	projectName := d.Get("project").(string)
	hostname := d.Get("hostname").(string)
	privateKey := d.Get("private_key").(string)

	organization := getOrganization(d, providerContext)
	if organization == "" {
		return errors.New("organization must be set at the project, or provider level")
	}

	privSigner, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return err
	}

	fingerprint := ssh.FingerprintLegacyMD5(privSigner.PublicKey())

	settings, err := providerClient.GetSettings(providerContext.VCS, organization, projectName)
	if err != nil {
		return err
	}

	for _, key := range settings.SSHKeys {
		if key.Fingerprint == fingerprint {
			return fmt.Errorf("SSH Key with fingerprint '%s' already exists for project '%s'", fingerprint, projectName)
		}
	}

	err = providerClient.AddSSHKey(providerContext.VCS, organization, projectName, hostname, privateKey)
	if err != nil {
		return err
	}

	d.SetId(generateSSHId(organization, projectName, fingerprint))
	d.Set("fingerprint", fingerprint)
	d.Set("hostname", hostname)

	return nil
}

func resourceCircleCISSHKeyRead(d *schema.ResourceData, meta interface{}) error {
	providerContext := meta.(ProviderContext)
	providerClient := providerContext.Client

	projectName := d.Get("project").(string)
	hostname := d.Get("hostname").(string)
	privateKey := d.Get("private_key").(string)

	organization := getOrganization(d, providerContext)
	if organization == "" {
		return errors.New("organization must be set at the project, or provider level")
	}

	privSigner, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return err
	}

	fingerprint := ssh.FingerprintLegacyMD5(privSigner.PublicKey())

	settings, err := providerClient.GetSettings(providerContext.VCS, organization, projectName)
	if err != nil {
		return err
	}

	for _, key := range settings.SSHKeys {
		if key.Fingerprint == fingerprint && key.Hostname == hostname {
			d.Set("fingerprint", fingerprint)
			d.Set("hostname", hostname)
			return nil
		}
	}

	return fmt.Errorf("Key with fingerprint '%s' not found in %s/%s/%s", fingerprint, providerContext.VCS, organization, projectName)
}

func resourceCircleCISSHKeyDelete(d *schema.ResourceData, meta interface{}) error {
	providerContext := meta.(ProviderContext)
	providerClient := providerContext.Client

	projectName := d.Get("project").(string)
	hostname := d.Get("hostname").(string)
	privateKey := d.Get("private_key").(string)

	organization := getOrganization(d, providerContext)
	if organization == "" {
		return errors.New("organization must be set at the project, or provider level")
	}

	privSigner, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return err
	}

	fingerprint := ssh.FingerprintLegacyMD5(privSigner.PublicKey())

	err = providerClient.DeleteSSHKey(providerContext.VCS, organization, projectName, hostname, fingerprint)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func resourceCircleCISSHKeyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	providerContext := meta.(ProviderContext)
	providerClient := providerContext.Client

	projectName := d.Get("project").(string)
	hostname := d.Get("hostname").(string)
	privateKey := d.Get("private_key").(string)

	organization := getOrganization(d, providerContext)
	if organization == "" {
		return false, errors.New("organization must be set at the project, or provider level")
	}

	privSigner, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return false, err
	}

	fingerprint := ssh.FingerprintLegacyMD5(privSigner.PublicKey())

	settings, err := providerClient.GetSettings(providerContext.VCS, organization, projectName)
	if err != nil {
		return false, err
	}

	for _, key := range settings.SSHKeys {
		if key.Fingerprint == fingerprint && key.Hostname == hostname {
			d.Set("fingerprint", fingerprint)
			d.Set("hostname", hostname)
			return true, nil
		}
	}

	return false, nil
}

func generateSSHId(organization, projectName, fingerprint string) string {
	vars := []string{
		organization,
		projectName,
		strings.ReplaceAll(fingerprint, ":", ""),
	}
	return strings.Join(vars, ".")
}
