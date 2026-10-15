package broken

// Deliberate type error: a string is not assignable to int. go/packages
// type-checks this package and reports the error, which is what the
// build_error path in route discovery must surface.
var _ int = "not an int"
