// Copyright 2015 Canonical Ltd.

package idp_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	gc "gopkg.in/check.v1"
	"gopkg.in/goose.v1/testing/httpsuite"
	"gopkg.in/goose.v1/testservices/identityservice"

	extidp "github.com/CanonicalLtd/blues-identity/idp"
	"github.com/CanonicalLtd/blues-identity/internal/idp"
	"github.com/CanonicalLtd/blues-identity/internal/mongodoc"
	"github.com/CanonicalLtd/blues-identity/params"
)

type keystoneSuite struct {
	idpSuite
	httpsuite.HTTPSuite
	idp     *idp.KeystoneIdentityProvider
	service *identityservice.UserPass
}

var _ = gc.Suite(&keystoneSuite{})

func (s *keystoneSuite) SetUpTest(c *gc.C) {
	s.idpSuite.SetUpTest(c)
	s.HTTPSuite.SetUpTest(c)
	s.service = identityservice.NewUserPass()
	s.service.SetupHTTP(s.Mux)
	s.idp = idp.NewKeystoneIdentityProvider(&extidp.KeystoneParams{
		Name:        "openstack",
		Description: "OpenStack",
		Domain:      "openstack",
		URL:         s.Server.URL,
	})
}

func (s *keystoneSuite) TearDownTest(c *gc.C) {
	s.HTTPSuite.TearDownTest(c)
	s.idpSuite.TearDownTest(c)
}

func (s *keystoneSuite) TestName(c *gc.C) {
	c.Assert(s.idp.Name(), gc.Equals, "openstack")
}

func (s *keystoneSuite) TestDescription(c *gc.C) {
	c.Assert(s.idp.Description(), gc.Equals, "OpenStack")
}

func (s *keystoneSuite) TestUseNameForDescription(c *gc.C) {
	provider := idp.NewKeystoneIdentityProvider(&extidp.KeystoneParams{
		Name: "openstack",
		URL:  s.Server.URL,
	})
	c.Assert(provider.Description(), gc.Equals, "openstack")
}

func (s *keystoneSuite) TestInteractive(c *gc.C) {
	c.Assert(s.idp.Interactive(), gc.Equals, true)
}

func (s *keystoneSuite) TestURL(c *gc.C) {
	tc := &testContext{}
	u, err := s.idp.URL(tc, "1")
	c.Assert(err, gc.IsNil)
	c.Assert(u, gc.Equals, "https://idp.test/login?waitid=1")
}

func (s *keystoneSuite) TestHandle(c *gc.C) {
	tc := &testContext{
		requestURL: "https://idp.test/login?waitid=1",
	}
	var err error
	tc.params.Request, err = http.NewRequest("GET", tc.requestURL, nil)
	c.Assert(err, gc.IsNil)
	rr := httptest.NewRecorder()
	tc.params.Response = rr
	s.idp.Handle(tc)
	c.Assert(rr.Code, gc.Equals, http.StatusOK)
	c.Assert(rr.HeaderMap.Get("Content-Type"), gc.Equals, "text/html;charset=UTF-8")
	c.Assert(rr.Body.String(), gc.Equals, `<!doctype html>
<html>
	<head><title>OpenStack Login</title></head>
	<body>
		<form method="POST" action="https://idp.test/login?waitid=1">
			<p><label>Username: <input type="text" name="username"></label></p>
			<p><label>Password: <input type="password" name="password"></label></p>
			<p><input type="submit"></p>
		</form>
	</body>
</html>
`)
}

func (s *keystoneSuite) TestHandleResponse(c *gc.C) {
	tc := &testContext{
		store:      s.store,
		requestURL: "https://idp.test/login?waitid=1",
		success:    true,
	}
	userInfo := s.service.AddUser("testuser", "testpass", "")
	v := url.Values{
		"username": []string{"testuser"},
		"password": []string{"testpass"},
	}
	var err error
	tc.params.Request, err = http.NewRequest("POST", tc.requestURL, strings.NewReader(v.Encode()))
	c.Assert(err, gc.IsNil)
	tc.params.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	tc.params.Response = rr
	s.idp.Handle(tc)
	c.Assert(tc.err, gc.IsNil)
	c.Assert(tc.macaroon, gc.Not(gc.IsNil))
	identity, err := s.store.GetIdentity(params.Username("testuser@openstack"))
	c.Assert(err, gc.IsNil)
	c.Assert(identity.ExternalID, gc.Equals, userInfo.Id+"@openstack")
	c.Assert(rr.Body.String(), gc.Equals, "login successful as user testuser@openstack\n")
}

func (s *keystoneSuite) TestHandleBadPassword(c *gc.C) {
	tc := &testContext{
		store:      s.store,
		requestURL: "https://idp.test/login?waitid=1",
	}
	s.service.AddUser("testuser", "testpass", "")
	v := url.Values{
		"username": []string{"testuser"},
		"password": []string{"nottestpass"},
	}
	var err error
	tc.params.Request, err = http.NewRequest("POST", tc.requestURL, strings.NewReader(v.Encode()))
	c.Assert(err, gc.IsNil)
	tc.params.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	tc.params.Response = rr
	s.idp.Handle(tc)
	c.Assert(tc.err, gc.ErrorMatches, `cannot log in: Unauthorised URL .*/tokens
caused by: request \(.*/tokens\) returned unexpected status: 401; error info: Failed: 401 error: Invalid user / password`)
	c.Assert(tc.macaroon, gc.IsNil)
}

func (s *keystoneSuite) TestHandleExistingUser(c *gc.C) {
	tc := &testContext{
		store:      s.store,
		requestURL: "https://idp.test/login?waitid=1",
		success:    true,
	}
	err := s.store.UpsertIdentity(&mongodoc.Identity{
		Username:   "testuser@openstack",
		ExternalID: "some other thing",
	})
	c.Assert(err, gc.IsNil)
	s.service.AddUser("testuser", "testpass", "")
	v := url.Values{
		"username": []string{"testuser"},
		"password": []string{"testpass"},
	}
	tc.params.Request, err = http.NewRequest("POST", tc.requestURL, strings.NewReader(v.Encode()))
	c.Assert(err, gc.IsNil)
	tc.params.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	tc.params.Response = rr
	s.idp.Handle(tc)
	c.Assert(tc.err, gc.ErrorMatches, "cannot update identity: cannot add user: duplicate username or external_id")
	c.Assert(tc.macaroon, gc.IsNil)
}
