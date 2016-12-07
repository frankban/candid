// Copyright 2016 Canonical Ltd.

package admincmd

import (
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/juju/cmd"
	"github.com/juju/gnuflag"
	"github.com/juju/idmclient"
	"github.com/juju/idmclient/params"
	"github.com/juju/persistent-cookiejar"
	"golang.org/x/net/publicsuffix"
	"gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
	"gopkg.in/macaroon-bakery.v2-unstable/httpbakery"
	"gopkg.in/macaroon-bakery.v2-unstable/httpbakery/agent"

	"github.com/CanonicalLtd/blues-identity/version"
)

// jujuLoggingConfigEnvKey matches osenv.JujuLoggingConfigEnvKey
// in the Juju project.
const jujuLoggingConfigEnvKey = "JUJU_LOGGING_CONFIG"

var cmdDoc = `
Manage the users on an identity server. By default the identity server
at https://api.jujucharms.com/identity will be modified. This can be
overridden either by setting the IDM_URL environment variable, or by
setting the --idm-url command line parameter.
`

func New() cmd.Command {
	supercmd := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:    "user-admin",
		Doc:     cmdDoc,
		Purpose: "manage users on an identity server",
		Log: &cmd.Log{
			DefaultConfig: os.Getenv(jujuLoggingConfigEnvKey),
		},
		Version: version.VersionInfo.Version,
	})
	supercmd.Register(newAddGroupCommand())
	supercmd.Register(newCreateAdminAgentCommand())
	supercmd.Register(newFindCommand())
	supercmd.Register(newRemoveGroupCommand())
	return supercmd
}

// idmCommand is a cmd.Command that provides a client for communicating
// with an identity manager. The identity manager can be sepcified via
// the command line, or using the IDM_URL environment variable.
type idmCommand struct {
	cmd.CommandBase

	url       string
	agentFile string

	// mu protects the fields below it.
	mu     sync.Mutex
	client *idmclient.Client
}

func (c *idmCommand) SetFlags(f *gnuflag.FlagSet) {
	c.CommandBase.SetFlags(f)
	f.StringVar(&c.url, "idm-url", "", "URL of the identity server (defaults to $IDM_URL)")
	f.StringVar(&c.agentFile, "a", "", "name of file containing agent login details")
	f.StringVar(&c.agentFile, "agent", "", "")
}

// Client creates a new idmclient.Client using the parameters specified
// in the flags and environment.
func (c *idmCommand) Client(ctxt *cmd.Context) (*idmclient.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return c.client, nil
	}
	bClient := httpbakery.NewClient()
	var err error
	bClient.Client.Jar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, errgo.Mask(err)
	}
	idmURL := idmURL(c.url)
	if c.agentFile != "" {
		a, err := Load(ctxt.AbsPath(c.agentFile))
		if err != nil {
			return nil, errgo.Notef(err, "cannot load agent information")
		}
		bClient.Key = &bakery.KeyPair{
			Public:  *a.PublicKey,
			Private: *a.PrivateKey,
		}
		u, err := url.Parse(idmURL)
		if err != nil {
			return nil, errgo.Notef(err, "invalid IDM URL")
		}
		agent.SetUpAuth(bClient, u, a.Username)
	} else {
		bClient.WebPageVisitor = httpbakery.WebBrowserVisitor
	}

	client, err := idmclient.New(idmclient.NewParams{
		BaseURL: idmURL,
		Client:  bClient,
	})
	if err != nil {
		return nil, errgo.Mask(err)
	}
	c.client = client
	return client, nil
}

func idmURL(url string) string {
	if url != "" {
		return url
	}
	if url := os.Getenv("IDM_URL"); url != "" {
		return url
	}
	return idmclient.Production
}

// usercmd is a cmd.Command that provides the ability to lookup and
// manipulate a user that is specified on the command line either by
// username or email address. Commands which wish to perform operations
// on a particular user should embed this type and use lookupUser to find
// the username to use in the subsequent requests.
type userCommand struct {
	idmCommand

	username string
	email    string
}

func (c *userCommand) SetFlags(f *gnuflag.FlagSet) {
	c.idmCommand.SetFlags(f)

	f.StringVar(&c.username, "u", "", "username of the user")
	f.StringVar(&c.username, "username", "", "")
	f.StringVar(&c.email, "e", "", "email address of the user")
	f.StringVar(&c.email, "email", "", "")
}

func (c *userCommand) Init(args []string) error {
	if c.username == "" && c.email == "" {
		return errgo.New("no user specified, please specify either username or email")
	} else if c.username != "" && c.email != "" {
		return errgo.New("both username and email specified, please specify either username or email")
	}
	return errgo.Mask(c.idmCommand.Init(args))
}

// AllowInterspersedFlags implements cmd.Command.AllowInterspersedFlags,
// by making them not allowed.
func (c *userCommand) AllowInterspersedFlags() bool {
	return false
}

func (c *userCommand) args() string {
	return "(-u username|-e email)"
}

func (c *userCommand) lookupUser(ctxt *cmd.Context) (params.Username, error) {
	if c.username != "" {
		return params.Username(c.username), nil
	}
	client, err := c.Client(ctxt)
	if err != nil {
		return "", errgo.Mask(err)
	}
	users, err := client.QueryUsers(&params.QueryUsersRequest{
		Email: c.email,
	})
	if err != nil {
		return "", errgo.Mask(err)
	}
	switch len(users) {
	case 0:
		return "", errgo.Newf("no user found for email %q", c.email)
	case 1:
		return params.Username(users[0]), nil
	}
	// Note: it is expected that for the most part this situation
	// should not come up as an identity server will not have many
	// identity providers and it is expected that they will not allow
	// more than one user to be registered with a unique email
	// address. There are however some situations in which this will
	// be possible. One case is when the user is a jujucharms.com
	// user and a snappy user which the identity server will keep
	// separate for implementation reasons, but could represent the
	// same Ubuntu SSO user.
	return "", errgo.Newf("more than one user found with email %q (%s)", c.email, strings.Join(users, ", "))
}
