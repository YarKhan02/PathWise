package gin

import (
	"github.com/YarKhan02/pathwise/internal/model"
)

// Facts converts a discovery result into the shared model's facts, tied to one
// scan. Routes become code_confirmed facts; misses become unverified facts that
// name why discovery stopped, so a gap in route coverage is recorded rather
// than mistaken for an application that has no such route.
func Facts(scanID string, res *Result) []model.Fact {
	facts := make([]model.Fact, 0, len(res.Routes)+len(res.Misses))

	for _, r := range res.Routes {
		loc := model.SourceLocation{File: r.Pos.Filename, StartLine: r.Pos.Line, EndLine: r.Pos.Line}
		subject := r.Subject()

		attrs := map[string]any{
			"method":  r.Method,
			"pattern": model.NormalizeRoute(r.Pattern),
			"handler": r.Handler,
		}
		if len(r.Middleware) > 0 {
			attrs["middleware"] = r.MiddlewareNames()
		}
		facts = append(facts, model.Fact{
			ScanID:   scanID,
			Kind:     model.FactRoute,
			Subject:  subject,
			Attrs:    attrs,
			Evidence: model.CodeConfirmed("gin route registration", loc),
		})

		if r.Handler != "" {
			facts = append(facts, model.Fact{
				ScanID:   scanID,
				Kind:     model.FactHandler,
				Subject:  r.Handler,
				Attrs:    map[string]any{"route": subject},
				Evidence: model.CodeConfirmed("handler bound to route "+subject, loc),
			})
		}

		// One middleware fact per entry, carrying its position in the chain and
		// the constant arguments it was built with, so Phase 3 can attach an
		// enforcement proof to the right link and a reader can see the role a
		// RequireRole("...") gate demands.
		for i, m := range r.Middleware {
			mattrs := map[string]any{"route": subject, "order": i}
			if len(m.Args) > 0 {
				mattrs["args"] = m.Args
			}
			facts = append(facts, model.Fact{
				ScanID:   scanID,
				Kind:     model.FactMiddleware,
				Subject:  m.Name,
				Attrs:    mattrs,
				Evidence: model.CodeConfirmed("middleware guarding route "+subject, loc),
			})
		}
	}

	for _, m := range res.Misses {
		loc := model.SourceLocation{File: m.Pos.Filename, StartLine: m.Pos.Line, EndLine: m.Pos.Line}
		facts = append(facts, model.Fact{
			ScanID:   scanID,
			Kind:     model.FactRoute,
			Subject:  "unresolved route at " + m.Pos.String(),
			Evidence: model.Unresolved(m.Reason, m.Detail, loc),
		})
	}

	return facts
}
