// Copyright 2014 Canonical Ltd.

package v1_test

import (
	"io"
	"net/http"
	"strings"

	"github.com/juju/testing/httptesting"
	gc "gopkg.in/check.v1"

	"github.com/CanonicalLtd/blues-identity/internal/mongodoc"
	"github.com/CanonicalLtd/blues-identity/params"
)

type usersSuite struct {
	apiSuite
}

var _ = gc.Suite(&usersSuite{})

func (s *usersSuite) TestUser(c *gc.C) {
	s.createUser(c, &params.User{
		UserName:   "jbloggs2",
		ExternalID: "http://example.com/jbloggs2",
		Email:      "jbloggs2@example.com",
		FullName:   "Joe Bloggs II",
		IDPGroups: []string{
			"test",
		},
	})
	s.createUser(c, &params.User{
		UserName:   "jbloggs3",
		ExternalID: "http://example.com/jbloggs3",
		Email:      "jbloggs3@example.com",
		FullName:   "Joe Bloggs III",
		IDPGroups: []string{
			"test",
		},
	})
	tests := []struct {
		about        string
		url          string
		method       string
		body         io.Reader
		username     string
		password     string
		expectStatus int
		expectBody   interface{}
	}{{
		about:  "create user",
		url:    apiURL("u/jbloggs"),
		method: "PUT",
		body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusOK,
		expectBody: params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		},
	}, {
		about:  "update existing user",
		url:    apiURL("u/jbloggs2"),
		method: "PUT",
		body: marshal(c, params.User{
			UserName:   "jbloggs2",
			ExternalID: "http://example.com/jbloggs2",
			Email:      "jbloggs2@example.com",
			FullName:   "Joe Bloggs II",
			IDPGroups: []string{
				"test",
				"test2",
			},
		}),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusOK,
		expectBody: params.User{
			UserName:   "jbloggs2",
			ExternalID: "http://example.com/jbloggs2",
			Email:      "jbloggs2@example.com",
			FullName:   "Joe Bloggs II",
			IDPGroups: []string{
				"test",
				"test2",
			},
		},
	}, {
		about:  "update existing username with different user",
		url:    apiURL("u/jbloggs2"),
		method: "PUT",
		body: marshal(c, params.User{
			UserName:   "jbloggs2",
			ExternalID: "http://example.com/joe.bloggs2",
			Email:      "jbloggs2@example.com",
			FullName:   "Joe Bloggs II",
			IDPGroups: []string{
				"test",
				"test2",
			},
		}),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusForbidden,
		expectBody: params.Error{
			Code:    "already exists",
			Message: `cannot store identity: cannot add user: duplicate username or external_id`,
		},
	}, {
		about:  "reuse external_id",
		url:    apiURL("u/jbloggs5"),
		method: "PUT",
		body: marshal(c, params.User{
			UserName:   "jbloggs5",
			ExternalID: "http://example.com/jbloggs2",
			Email:      "jbloggs5@example.com",
			FullName:   "Joe Bloggs V",
			IDPGroups: []string{
				"test",
			},
		}),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusForbidden,
		expectBody: params.Error{
			Code:    "already exists",
			Message: `cannot store identity: cannot add user: duplicate username or external_id`,
		},
	}, {
		about:        "known user",
		url:          apiURL("u/jbloggs3"),
		method:       "GET",
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusOK,
		expectBody: params.User{
			UserName:   "jbloggs3",
			ExternalID: "http://example.com/jbloggs3",
			Email:      "jbloggs3@example.com",
			FullName:   "Joe Bloggs III",
			IDPGroups: []string{
				"test",
			},
		},
	}, {
		about:        "unknown user",
		url:          apiURL("u/jbloggs4"),
		method:       "GET",
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusNotFound,
		expectBody: params.Error{
			Code:    "not found",
			Message: `user "jbloggs4" not found: not found`,
		},
	}, {
		about:        "get no username",
		url:          apiURL("u/"),
		method:       "GET",
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusNotFound,
		expectBody: params.Error{
			Code:    "not found",
			Message: `user "" not found: not found`,
		},
	}, {
		about:    "unsupported method",
		url:      apiURL("u/jbloggs"),
		method:   "POST",
		username: adminUsername,
		password: adminPassword,
		body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		expectStatus: http.StatusBadRequest,
		expectBody: params.Error{
			Code:    "bad request",
			Message: "unsupported method \"POST\"",
		},
	}, {
		about:    "put no userid",
		url:      apiURL("u/"),
		method:   "PUT",
		username: adminUsername,
		password: adminPassword,
		body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		expectStatus: http.StatusBadRequest,
		expectBody: params.Error{
			Code:    "bad request",
			Message: "illegal username: \"\"",
		},
	}, {
		about:    "put userid mismatch",
		url:      apiURL("u/jbloggs6"),
		method:   "PUT",
		username: adminUsername,
		password: adminPassword,
		body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs6",
			Email:      "jbloggs6@example.com",
			FullName:   "Joe Bloggs VI",
			IDPGroups: []string{
				"test6",
			},
		}),
		expectStatus: http.StatusOK,
		expectBody: params.User{
			UserName:   "jbloggs6",
			ExternalID: "http://example.com/jbloggs6",
			Email:      "jbloggs6@example.com",
			FullName:   "Joe Bloggs VI",
			IDPGroups: []string{
				"test6",
			},
		},
	}, {
		about:        "bad json",
		url:          apiURL("u/jbloggs2"),
		method:       "PUT",
		username:     adminUsername,
		password:     adminPassword,
		body:         strings.NewReader("invalid"),
		expectStatus: http.StatusBadRequest,
		expectBody: params.Error{
			Code:    "bad request",
			Message: "invalid JSON data: invalid character 'i' looking for beginning of value",
		},
	}, {
		about:    "incorrect username",
		url:      apiURL("u/jbloggs2"),
		method:   "PUT",
		username: "bad user",
		password: adminPassword,
		body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		expectStatus: http.StatusUnauthorized,
		expectBody: params.Error{
			Code:    "unauthorized",
			Message: "invalid credentials",
		},
	}, {
		about:    "incorrect password",
		url:      apiURL("u/jbloggs2"),
		method:   "PUT",
		username: adminUsername,
		password: "bad password",
		body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		expectStatus: http.StatusUnauthorized,
		expectBody: params.Error{
			Code:    "unauthorized",
			Message: "invalid credentials",
		},
	}, {
		about:  "no credentials",
		url:    apiURL("u/jbloggs2"),
		method: "PUT",
		body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		expectStatus: http.StatusUnauthorized,
		expectBody: params.Error{
			Code:    "unauthorized",
			Message: "unauthorized: invalid or missing HTTP auth header",
		},
	}, {
		about:  "bad username",
		url:    apiURL("u/jbloggs{}"),
		method: "PUT",
		body: marshal(c, params.User{
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusBadRequest,
		expectBody: params.Error{
			Code:    "bad request",
			Message: `illegal username: "jbloggs{}"`,
		},
	}, {
		about:  "long username",
		url:    apiURL("u/jbloggs001jbloggs002jbloggs003jbloggs004jbloggs005jbloggs006jbloggs007jbloggs008jbloggs009jbloggs010jbloggs011jbloggs012jbloggs013jbloggs014jbloggs015jbloggs016jbloggs017jbloggs018jbloggs019jbloggs020jbloggs021jbloggs022jbloggs023jbloggs024jbloggs025jbloggs026"),
		method: "PUT",
		body: marshal(c, params.User{
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusBadRequest,
		expectBody: params.Error{
			Code:    "bad request",
			Message: `username longer than 256 characters`,
		},
	}, {
		about:  "invalid subpath",
		url:    apiURL("u/jbloggs2/notthere"),
		method: "GET",
		body: marshal(c, params.User{
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusNotFound,
		expectBody: params.Error{
			Code:    "not found",
			Message: `/notthere not found`,
		},
	}, {
		about:  "no external_id",
		url:    apiURL("u/jbloggs8"),
		method: "PUT",
		body: marshal(c, params.User{
			Email:    "jbloggs8@example.com",
			FullName: "Joe Bloggs VIII",
			IDPGroups: []string{
				"test",
			},
		}),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusBadRequest,
		expectBody: params.Error{
			Code:    "bad request",
			Message: `external_id not specified`,
		},
	}}
	for i, test := range tests {
		c.Logf("%d. %s", i, test.about)
		httptesting.AssertJSONCall(c, httptesting.JSONCallParams{
			Handler: s.srv,
			URL:     test.url,
			Method:  test.method,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body:         test.body,
			Username:     test.username,
			Password:     test.password,
			ExpectStatus: test.expectStatus,
			ExpectBody:   test.expectBody,
		})
	}
}

func (s *usersSuite) TestCreateUserWritesToDatabase(c *gc.C) {
	httptesting.AssertJSONCall(c, httptesting.JSONCallParams{
		Handler: s.srv,
		URL:     apiURL("u/jbloggs"),
		Method:  "PUT",
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		Username:     adminUsername,
		Password:     adminPassword,
		ExpectStatus: http.StatusOK,
		ExpectBody: params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		},
	})
	var doc mongodoc.Identity
	err := s.store.DB.Identities().Find(nil).One(&doc)
	c.Assert(err, gc.IsNil)
	c.Assert(doc.UserName, gc.Equals, "jbloggs")
	c.Assert(doc.ExternalID, gc.Equals, "http://example.com/jbloggs")
	c.Assert(doc.Email, gc.Equals, "jbloggs@example.com")
	c.Assert(doc.FullName, gc.Equals, "Joe Bloggs")
	c.Assert(doc.Groups, gc.DeepEquals, []string{"test"})
}

func (s *usersSuite) TestQueryUsers(c *gc.C) {
	s.createUser(c, &params.User{
		UserName:   "jbloggs2",
		ExternalID: "http://example.com/jbloggs2",
		Email:      "jbloggs2@example.com",
		FullName:   "Joe Bloggs II",
		IDPGroups: []string{
			"test",
		},
	},
	)
	tests := []struct {
		about        string
		url          string
		method       string
		body         io.Reader
		username     string
		password     string
		expectStatus int
		expectBody   interface{}
	}{{
		about:        "query existing user",
		url:          apiURL("u?external_id=http://example.com/jbloggs2"),
		method:       "GET",
		body:         nil,
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusOK,
		expectBody:   []string{"jbloggs2"},
	}, {
		about:        "query non-existing user",
		url:          apiURL("u?external_id=http://example.com/jbloggs"),
		method:       "GET",
		body:         nil,
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusOK,
		expectBody:   []string{},
	}, {
		about:        "no query parameter",
		url:          apiURL("u"),
		method:       "GET",
		body:         nil,
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusOK,
		expectBody:   []string{"jbloggs2"},
	}, {
		about:        "incorrect method",
		url:          apiURL("u?external_id=http://example.com/jbloggs"),
		method:       "DELETE",
		body:         nil,
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusBadRequest,
		expectBody: params.Error{
			Code:    "bad request",
			Message: "unsupported method \"DELETE\"",
		},
	}, {
		about:    "incorrect username",
		url:      apiURL("u?external_id=http://example.com/jbloggs2"),
		method:   "GET",
		username: "bad user",
		password: adminPassword,
		body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		expectStatus: http.StatusUnauthorized,
		expectBody: params.Error{
			Code:    "unauthorized",
			Message: "invalid credentials",
		},
	}, {
		about:    "incorrect password",
		url:      apiURL("u?external_id=http://example.com/jbloggs2"),
		method:   "GET",
		username: adminUsername,
		password: "bad password",
		body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		expectStatus: http.StatusUnauthorized,
		expectBody: params.Error{
			Code:    "unauthorized",
			Message: "invalid credentials",
		},
	}, {
		about:  "no credentials",
		url:    apiURL("u?external_id=http://example.com/jbloggs2"),
		method: "GET",
		body: marshal(c, params.User{
			UserName:   "jbloggs",
			ExternalID: "http://example.com/jbloggs",
			Email:      "jbloggs@example.com",
			FullName:   "Joe Bloggs",
			IDPGroups: []string{
				"test",
			},
		}),
		expectStatus: http.StatusUnauthorized,
		expectBody: params.Error{
			Code:    "unauthorized",
			Message: "unauthorized: invalid or missing HTTP auth header",
		},
	}}
	for i, test := range tests {
		c.Logf("%d. %s", i, test.about)
		httptesting.AssertJSONCall(c, httptesting.JSONCallParams{
			Handler: s.srv,
			URL:     test.url,
			Method:  test.method,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body:         test.body,
			Username:     test.username,
			Password:     test.password,
			ExpectStatus: test.expectStatus,
			ExpectBody:   test.expectBody,
		})
	}
}

func (s *usersSuite) TestUserIDPGroups(c *gc.C) {
	s.createUser(c, &params.User{
		UserName:   "test",
		ExternalID: "http://example.com/test",
		Email:      "test@example.com",
		FullName:   "Test User",
		IDPGroups:  []string{},
	})
	s.createUser(c, &params.User{
		UserName:   "test2",
		ExternalID: "http://example.com/test2",
		Email:      "test2@example.com",
		FullName:   "Test User II",
		IDPGroups: []string{
			"test",
		},
	})
	s.createUser(c, &params.User{
		UserName:   "test3",
		ExternalID: "http://example.com/test3",
		Email:      "test3@example.com",
		FullName:   "Test User III",
		IDPGroups: []string{
			"test",
			"test2",
		},
	})
	tests := []struct {
		about        string
		url          string
		username     string
		password     string
		expectStatus int
		expectBody   interface{}
	}{{
		about:        "user without groups",
		url:          apiURL("u/test/idpgroups"),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusOK,
		expectBody:   []string{},
	}, {
		about:        "user with 1 group",
		url:          apiURL("u/test2/idpgroups"),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusOK,
		expectBody:   []string{"test"},
	}, {
		about:        "user with 2 groups",
		url:          apiURL("u/test3/idpgroups"),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusOK,
		expectBody:   []string{"test", "test2"},
	}, {
		about:        "unknown user",
		url:          apiURL("u/test4/idpgroups"),
		username:     adminUsername,
		password:     adminPassword,
		expectStatus: http.StatusNotFound,
		expectBody: params.Error{
			Code:    "not found",
			Message: `user "test4" not found: not found`,
		},
	}}
	for i, test := range tests {
		c.Logf("%d. %s", i, test.about)
		httptesting.AssertJSONCall(c, httptesting.JSONCallParams{
			Handler:      s.srv,
			URL:          test.url,
			Username:     test.username,
			Password:     test.password,
			ExpectStatus: test.expectStatus,
			ExpectBody:   test.expectBody,
		})
	}
}

func (s *usersSuite) createUser(c *gc.C, user *params.User) {
	httptesting.AssertJSONCall(c, httptesting.JSONCallParams{
		Handler: s.srv,
		URL:     apiURL("u/" + user.UserName),
		Method:  "PUT",
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body:         marshal(c, user),
		Username:     adminUsername,
		Password:     adminPassword,
		ExpectStatus: http.StatusOK,
		ExpectBody:   user,
	})
}