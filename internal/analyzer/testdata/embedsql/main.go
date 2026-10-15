// A fixture mirroring AcademyEngine's SQL layout: the query text lives in a
// .sql file pulled in with //go:embed, so at the call site the query argument
// is a package-level string variable, not a literal. The analyzer must resolve
// the embed to recover the table the query touches.
package main

import (
	"context"
	"database/sql"
	_ "embed"
)

//go:embed sql/widget_by_id.sql
var widgetByIDSQL string

var db *sql.DB

func loadWidget(ctx context.Context, id string) {
	_, _ = db.QueryContext(ctx, widgetByIDSQL, id)
}

func main() {
	loadWidget(context.Background(), "1")
}
