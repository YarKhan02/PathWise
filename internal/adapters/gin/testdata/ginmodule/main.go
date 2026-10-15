// A fixture mirroring AcademyEngine's real shape: routes are registered inside
// a method that receives the engine as a parameter (RegisterRoutes(r
// *gin.Engine)), with the group's auth middleware attached at r.Group(...), and
// handlers bound as methods on a Handler struct. The engine itself is built in
// a different function (main), so within RegisterRoutes the router is only a
// parameter. The adapter must still recover the routes.
package main

import "github.com/gin-gonic/gin"

type Handler struct{}

func (h *Handler) GetCourse(c *gin.Context)    {}
func (h *Handler) InsertCourse(c *gin.Context) {}

type Module struct{ h *Handler }

func Auth() gin.HandlerFunc                   { return func(c *gin.Context) {} }
func RequireRole(role string) gin.HandlerFunc { return func(c *gin.Context) {} }

// RegisterRoutes is a method, and r arrives as a parameter — the pattern a
// Members-only, gin.New()-anchored walk cannot resolve.
func (m *Module) RegisterRoutes(r *gin.Engine) {
	course := r.Group("/course", Auth(), RequireRole("admin"))
	course.GET("", m.h.GetCourse)
	course.POST("", m.h.InsertCourse)
}

func main() {
	r := gin.New()
	m := &Module{h: &Handler{}}
	m.RegisterRoutes(r)
	_ = r.Run()
}
