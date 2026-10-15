package report

import (
	"html/template"
	"io"
	"strings"
)

// RenderHTML writes the report as one self-contained HTML page: no server, no
// external assets, openable straight from disk. It groups the data around the
// question a reader asks — pick a role, see which endpoints it reaches and the
// tables behind them.
func RenderHTML(w io.Writer, r *Report) error {
	return pageTemplate.Execute(w, r)
}

var pageTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"join":  strings.Join,
	"lower": strings.ToLower,
	"rolesOf": func(e Endpoint) []string {
		if len(e.Roles) == 0 {
			return []string{"public"}
		}
		return e.Roles
	},
	"searchText": func(e Endpoint) string {
		parts := []string{e.Method, e.Path, e.Handler}
		for _, res := range e.Resources {
			parts = append(parts, res.Table)
		}
		parts = append(parts, e.Roles...)
		return strings.ToLower(strings.Join(parts, " "))
	},
}).Parse(pageHTML))

const pageHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>PathWise — {{.Project}}</title>
<style>
:root{
  --bg:#f7f8fa; --panel:#fff; --ink:#1b1f24; --muted:#6b7480; --line:#e3e7ec;
  --accent:#3b5bdb; --chip:#eef1f6; --role:#e7edff; --role-ink:#2a3f9d;
  --get:#2b8a3e; --post:#1971c2; --put:#e8590c; --patch:#e8590c; --delete:#c92a2a; --other:#5c636e;
  --warn:#b5830a; --warnbg:#fff7e0; --find:#c92a2a; --findbg:#ffe9e9;
}
:root:not([data-theme="light"]){@media (prefers-color-scheme:dark){
  --bg:#0f1216; --panel:#171b21; --ink:#e6e9ee; --muted:#9aa4b0; --line:#262c34;
  --accent:#748ffc; --chip:#222932; --role:#20304f; --role-ink:#aec4ff;
  --warnbg:#2a2412; --findbg:#2a1717;
}}
:root[data-theme="dark"]{
  --bg:#0f1216; --panel:#171b21; --ink:#e6e9ee; --muted:#9aa4b0; --line:#262c34;
  --accent:#748ffc; --chip:#222932; --role:#20304f; --role-ink:#aec4ff;
  --warnbg:#2a2412; --findbg:#2a1717;
}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--ink);font:15px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;padding:0 16px}
.wrap{max-width:1100px;margin:0 auto;padding:28px 0 64px}
h1{font-size:20px;margin:0 0 2px}
.sub{color:var(--muted);font-size:13px;margin-bottom:20px}
.stats{display:flex;flex-wrap:wrap;gap:10px;margin-bottom:22px}
.stat{background:var(--panel);border:1px solid var(--line);border-radius:10px;padding:10px 14px;min-width:92px}
.stat b{display:block;font-size:22px;line-height:1.2}
.stat span{color:var(--muted);font-size:12px}
.controls{display:flex;flex-wrap:wrap;gap:8px;align-items:center;margin-bottom:14px}
.roles{display:flex;flex-wrap:wrap;gap:6px}
.rolebtn{border:1px solid var(--line);background:var(--panel);color:var(--ink);border-radius:999px;padding:5px 12px;font-size:13px;cursor:pointer}
.rolebtn.active{background:var(--accent);color:#fff;border-color:var(--accent)}
#search{flex:1;min-width:160px;background:var(--panel);border:1px solid var(--line);color:var(--ink);border-radius:8px;padding:8px 11px;font-size:14px}
table{width:100%;border-collapse:collapse;background:var(--panel);border:1px solid var(--line);border-radius:12px;overflow:hidden}
th,td{text-align:left;padding:10px 12px;border-bottom:1px solid var(--line);vertical-align:top}
th{font-size:12px;text-transform:uppercase;letter-spacing:.04em;color:var(--muted);font-weight:600}
tr:last-child td{border-bottom:none}
.method{font-weight:700;font-size:12px;padding:2px 7px;border-radius:6px;color:#fff;display:inline-block}
.m-get{background:var(--get)}.m-post{background:var(--post)}.m-put{background:var(--put)}.m-patch{background:var(--patch)}.m-delete{background:var(--delete)}.m-other{background:var(--other)}
.path{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:13px}
.chip{display:inline-block;background:var(--chip);border-radius:6px;padding:2px 7px;font-size:12px;margin:2px 3px 2px 0}
.role{background:var(--role);color:var(--role-ink);font-weight:600}
.ops{color:var(--muted);font-size:11px}
.badge{font-size:11px;padding:2px 7px;border-radius:6px;font-weight:600}
.unres{background:var(--warnbg);color:var(--warn)}
.find{background:var(--findbg);color:var(--find)}
.handler{color:var(--muted);font-size:11px;font-family:ui-monospace,monospace;display:block;margin-top:3px}
.empty{color:var(--muted);font-size:13px;padding:24px;text-align:center}
@media (max-width:640px){.hide-sm{display:none}}
</style>
</head>
<body>
<div class="wrap">
  <h1>PathWise — {{.Project}}</h1>
  <div class="sub">scan {{.ScanID}} · schema {{.SchemaSource}} · generated {{.GeneratedAt}}</div>

  <div class="stats">
    <div class="stat"><b>{{.RouteCount}}</b><span>endpoints</span></div>
    <div class="stat"><b>{{.RoleCount}}</b><span>roles</span></div>
    <div class="stat"><b>{{.TableCount}}</b><span>tables</span></div>
    <div class="stat"><b>{{.FindingCount}}</b><span>findings</span></div>
    <div class="stat"><b>{{.UnresolvedCount}}</b><span>unresolved queries</span></div>
  </div>

  <div class="controls">
    <div class="roles" id="roles">
      <button class="rolebtn active" data-role="">All</button>
      {{range .Roles}}<button class="rolebtn" data-role="{{.}}">{{.}}</button>{{end}}
      <button class="rolebtn" data-role="public">public</button>
    </div>
    <input id="search" type="search" placeholder="filter by path, handler, table…">
  </div>

  <table id="tbl">
    <thead><tr>
      <th>Method</th><th>Endpoint</th><th>Roles</th><th>Resources</th><th class="hide-sm">Status</th>
    </tr></thead>
    <tbody>
    {{range .Endpoints}}
      <tr data-roles="{{join (rolesOf .) " "}}" data-text="{{searchText .}}">
        <td><span class="method m-{{lower .Method}}">{{.Method}}</span></td>
        <td><span class="path">{{.Path}}</span>{{if .Handler}}<span class="handler">{{.Handler}}</span>{{end}}</td>
        <td>{{range rolesOf .}}<span class="chip role">{{.}}</span>{{end}}</td>
        <td>
          {{range .Resources}}<span class="chip">{{.Table}} <span class="ops">{{join .Operations ","}}</span></span>{{end}}
          {{if and (not .Resources) (not .Unresolved)}}<span class="ops">—</span>{{end}}
        </td>
        <td class="hide-sm">
          {{if .Unresolved}}<span class="badge unres">dynamic SQL</span>{{end}}
          {{range .Findings}}<span class="badge find" title="{{.Title}}: {{.Detail}}">{{.Kind}}</span>{{end}}
        </td>
      </tr>
    {{end}}
    </tbody>
  </table>
  <div class="empty" id="empty" style="display:none">No endpoints match.</div>
</div>

<script>
(function(){
  var rows = Array.prototype.slice.call(document.querySelectorAll('#tbl tbody tr'));
  var role = "";
  var q = "";
  function apply(){
    var shown = 0;
    rows.forEach(function(tr){
      var roles = (" "+tr.getAttribute('data-roles')+" ");
      var text = tr.getAttribute('data-text');
      var okRole = role === "" || roles.indexOf(" "+role+" ") >= 0;
      var okText = q === "" || text.indexOf(q) >= 0;
      var show = okRole && okText;
      tr.style.display = show ? "" : "none";
      if(show) shown++;
    });
    document.getElementById('empty').style.display = shown ? "none" : "";
  }
  document.getElementById('roles').addEventListener('click', function(e){
    var b = e.target.closest('.rolebtn'); if(!b) return;
    role = b.getAttribute('data-role');
    document.querySelectorAll('.rolebtn').forEach(function(x){ x.classList.toggle('active', x===b); });
    apply();
  });
  document.getElementById('search').addEventListener('input', function(e){
    q = e.target.value.toLowerCase().trim(); apply();
  });
})();
</script>
</body>
</html>
`
