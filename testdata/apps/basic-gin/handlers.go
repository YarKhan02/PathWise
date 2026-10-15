package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetStudent reads a student by a caller-supplied id with no ownership
// constraint of any kind: no ownership check in the handler, and no
// identity-derived predicate in the query.
//
// pathwise:expect kind=missing_ownership_check id=idor-get-student
func GetStudent(c *gin.Context) {
	id := c.Param("id")

	s, err := studentByID(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, s)
}

// GetStudentSafe is the correct form of the handler above. The failure branch
// of CanAccessStudent aborts, and that abort dominates the response write, so
// no path reaches c.JSON without the check having passed.
//
// pathwise:safe id=ownership-enforced
func GetStudentSafe(c *gin.Context) {
	user := CurrentUser(c)
	id := c.Param("id")

	s, err := studentByID(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if !CanAccessStudent(user, s) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	c.JSON(http.StatusOK, s)
}

// GetStudentProfile enforces ownership correctly but serializes national_id,
// which is declared sensitive and is not in this endpoint's declared scope.
// The authorization is sound; the response is over-broad.
//
// Named GetStudentProfile, not StudentProfile, because the row shape it
// serializes is the StudentProfile type in store.go; a handler and a type
// cannot share a name in one package.
//
// pathwise:expect kind=response_field_exceeds_scope id=scope-profile-national-id
func GetStudentProfile(c *gin.Context) {
	user := CurrentUser(c)
	id := c.Param("id")

	p, err := studentProfileByID(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if user == nil || user.OrgID != p.OrgID {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	c.JSON(http.StatusOK, p)
}

// ListBatches reads every batch with no predicate on the tenant
// discriminator, so a caller in one organization reads another's rows.
//
// pathwise:expect kind=missing_tenant_filter id=tenant-list-batches
func ListBatches(c *gin.Context) {
	batches, err := allBatches(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, batches)
}

// AdminReport calls a role check and discards its result. The call is present,
// so a name-matching analyzer would believe the endpoint is protected; the
// value is never branched on, so it protects nothing.
//
// The admin group's RequireRole middleware does guard this route, which is
// deliberate: the finding is about the handler's own dead check, and a correct
// analyzer must attribute it there rather than reporting the endpoint
// unprotected.
//
// pathwise:expect kind=role_check_unenforced id=role-discarded-admin-report
func AdminReport(c *gin.Context) {
	user := CurrentUser(c)

	_ = IsAdmin(user)

	rows, err := reportRows(c.Request.Context(), user.OrgID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, rows)
}

// TeacherDashboard performs a role check inside a conditional, so the check's
// failure branch does not dominate the response write: when strict is absent
// the handler responds without ever evaluating the role.
//
// This is the dominance case rather than the discarded-value case, and an
// analyzer that only looks for ignored return values will miss it.
//
// pathwise:expect kind=role_check_unenforced id=role-nondominating-dashboard
func TeacherDashboard(c *gin.Context) {
	user := CurrentUser(c)

	if c.Query("strict") == "1" {
		if !HasRole(user, "teacher") {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}

	rows, err := dashboardRows(c.Request.Context(), user.OrgID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, rows)
}

// SearchStudents builds its query text at runtime, so the predicate set is
// not recoverable from source. This is a negative control: the correct
// behavior is to record an unverified fact naming dynamic_sql. Reporting a
// finding here would be a false positive, and reporting nothing at all would
// be a silent gap.
//
// pathwise:unresolved reason=dynamic_sql id=dynamic-sql-search
func SearchStudents(c *gin.Context) {
	term := c.Query("q")

	query := fmt.Sprintf(
		"SELECT id, name FROM students WHERE name LIKE '%%%s%%' LIMIT 50", term)

	students, err := rawStudentQuery(c.Request.Context(), query)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, students)
}

// Login is unauthenticated by design. A correct analyzer must not report a
// missing authorization check on an endpoint that has no principal to check.
//
// pathwise:safe id=login-public-by-design
func Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	token, err := createSession(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
