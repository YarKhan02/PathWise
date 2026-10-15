// A deliberately broken module: it does not type-check. It exists so a test
// can prove that route discovery reports a build failure as a build_error fact
// rather than as an empty, all-clear result. Its own module keeps it out of the
// parent build graph.
module brokentest

go 1.26
