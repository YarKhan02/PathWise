// A tiny Gin app that registers one route with handlers spread from a slice a
// function returns, and one ordinary route. The spread form hides the handler
// from the adapter's value tracking, so discovery must record it as an
// unresolved route rather than emit it with an empty handler. The ordinary
// route confirms discovery still works alongside the miss.
package main

import "github.com/gin-gonic/gin"

func mw() gin.HandlerFunc { return func(c *gin.Context) {} }

func show(c *gin.Context) {}

// handlers returns the chain by value, so at the call site the variadic
// argument is a function result, not a slice literal the adapter can see into.
func handlers() []gin.HandlerFunc { return []gin.HandlerFunc{mw(), show} }

func main() {
	r := gin.New()
	r.GET("/spread", handlers()...)
	r.GET("/plain", show)
	_ = r.Run()
}
