package circleci

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestAccCircleCIAddSSHKeyThenDelete(t *testing.T) {
	organization := "ZymoticB"
	project := "cci-tfp-test-repo"
	resourceName := "circleci_ssh_key.key1"
	hostname := "github.com"

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	require.Nil(t, err)

	privateKeyDERBytes := x509.MarshalPKCS1PrivateKey(priv)

	var encodedPem bytes.Buffer
	err = pem.Encode(&encodedPem, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDERBytes,
	})
	require.Nil(t, err)

	privateKey := encodedPem.String()

	sshPriv, err := ssh.NewSignerFromSigner(priv)
	require.Nil(t, err)

	fingerprint := ssh.FingerprintLegacyMD5(sshPriv.PublicKey())

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOrgProviders,
		CheckDestroy: testAccCircleCISSHKeyCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCircleCISSHKey(organization, project, hostname, privateKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", fmt.Sprintf("%s.%s.%s", organization, project, strings.ReplaceAll(fingerprint, ":", ""))),
					resource.TestCheckResourceAttr(resourceName, "hostname", hostname),
					resource.TestCheckResourceAttr(resourceName, "fingerprint", fingerprint),
				),
			},
		},
	})
}

func testAccCircleCISSHKeyCheckDestroy(s *terraform.State) error {
	ctx := testAccOrgProvider.Meta().(ProviderContext)
	return projectCheckDestroy(ctx, s)
}

func sshKeyCheckDestroy(ctx ProviderContext, s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "circleci_ssh_key" {
			continue
		}

		organization := rs.Primary.Attributes["organization"]
		if organization == "" {
			organization = ctx.Org
		}

		hostname := rs.Primary.Attributes["hostname"]
		fingerprint := rs.Primary.Attributes["fingerprint"]
		project := rs.Primary.Attributes["project"]

		settings, err := ctx.Client.GetSettings(ctx.VCS, organization, project)
		if err != nil {
			return err
		}

		for _, key := range settings.SSHKeys {
			if key.Hostname == hostname && key.Fingerprint == fingerprint {
				return fmt.Errorf("ssh key for hostname: %s with fingerprint: %s should have been deleted", hostname, fingerprint)
			}
		}
	}

	return nil
}

func testAccCircleCISSHKey(org, project, hostname, privateKey string) string {
	return fmt.Sprintf(`
resource "circleci_ssh_key" "key1" {
  organization = "%s"
  project = "%s"
  hostname = "%s"
  private_key = <<-EOF
%s
EOF
}
`, org, project, hostname, privateKey)
}
