// Corpus app: basic-gin.
//
// A deliberately small Gin + database/sql application whose authorization
// structure is known in advance. It seeds one instance of each finding kind
// PathWise reports, plus negative controls: correctly-authorized handlers
// that must not produce findings, and a dynamically-built query that must be
// reported as unverified rather than either passing or failing.
//
// Expectations live in expected.yaml, anchored to the pathwise: marker
// comments in these files.
package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

var db *sql.DB

func main() {
	r := gin.New()
	r.Use(gin.Recovery())

	// Public: no authentication at all.
	r.POST("/login", Login)

	// Authenticated routes. RequireAuth populates the principal that the
	// handlers below read via CurrentUser.
	api := r.Group("/", RequireAuth)
	{
		api.GET("/students/:id", GetStudent)
		api.GET("/students/:id/safe", GetStudentSafe)
		api.GET("/students/:id/profile", GetStudentProfile)
		api.GET("/search/students", SearchStudents)
		api.GET("/batches", ListBatches)
		api.GET("/dashboard", TeacherDashboard)
	}

	// An admin group whose middleware does enforce a role, so that the
	// unenforced check inside AdminReport is attributable to the handler
	// rather than to a missing middleware.
	admin := r.Group("/admin", RequireAuth, RequireRole("admin"))
	{
		admin.GET("/reports", AdminReport)
	}

	srv := &http.Server{Addr: "127.0.0.1:8080", Handler: r}
	log.Fatal(srv.ListenAndServe())
}
