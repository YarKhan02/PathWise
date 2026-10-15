package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// User is the authenticated principal. OrgID is the tenant discriminator.
type User struct {
	ID    int64
	OrgID int64
	Role  string
}

const principalKey = "principal"

// RequireAuth resolves the principal and aborts when absent. This is the
// authentication boundary, not an authorization check: it establishes who the
// caller is and says nothing about what they may reach.
func RequireAuth(c *gin.Context) {
	u, err := principalFromToken(c.GetHeader("Authorization"))
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	c.Set(principalKey, u)
	c.Next()
}

// RequireRole enforces a role at the middleware layer. The abort dominates
// every downstream handler, which is what makes this enforcement rather than
// a mere call.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil || u.Role != role {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

// CurrentUser returns the authenticated principal, or nil.
func CurrentUser(c *gin.Context) *User {
	v, ok := c.Get(principalKey)
	if !ok {
		return nil
	}
	u, _ := v.(*User)
	return u
}

// CanAccessStudent is the ownership predicate: a teacher may reach a student
// only within their own organization.
func CanAccessStudent(u *User, s *Student) bool {
	if u == nil || s == nil {
		return false
	}
	return u.OrgID == s.OrgID
}

// HasRole reports whether the principal holds a role.
func HasRole(u *User, role string) bool {
	return u != nil && u.Role == role
}

// IsAdmin reports whether the principal is an administrator.
func IsAdmin(u *User) bool {
	return HasRole(u, "admin")
}

// principalFromToken is a stand-in for real token verification.
func principalFromToken(token string) (*User, error) {
	if token == "" {
		return nil, errUnauthorized
	}
	return lookupSession(token)
}
