// Package version carries the analyzer version. It is recorded on every scan
// so that historical findings can be interpreted against the analyzer that
// produced them: a finding absent from a later scan may mean the code was
// fixed, or merely that the analyzer changed.
package version

// Analyzer is the semantic version of the analysis engine. Bump the minor
// component whenever analysis results could change for unchanged source.
const Analyzer = "0.1.0-phase0"
