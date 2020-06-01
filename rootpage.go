package stdhttp

//----------------------------------------------------------------------------------------------------------------------------//

var rootPage = `<!DOCTYPE html>
<html lang="en">
	<head>
		<title>{{.Name}}</title>
		<meta charset="UTF-8" />
		<link rel="stylesheet" href="/___.css" />
	</head>

	<body>
		<h4><img src="/favicon.ico" style="width: 16px; height: 16px; position: relative; top: 2px;" alt="" />&nbsp;<em>{{.Name}} [{{.App}} {{.Version}}{{if .Tags}}&nbsp;{{.Tags}}{{end}}]</em></h4>

		{{if .ErrMsg}}<p><strong class="attention">{{.ErrMsg}}</strong></p>{{end}}

		<h6>Logging level</h6>
		<table class="grd">
		{{range $_, $CurrentLogLevel := .LogLevels}}
			<tr>
				<th class="left">
					{{if index $CurrentLogLevel 0}}{{index $CurrentLogLevel 0}}{{else}}default{{end}}
				</th>
				{{range $_, $LevelName := $.LogLevelNames}}
					<td>
						<a href="/set-log-level?facility={{index $CurrentLogLevel 0}}&amp;level={{$LevelName}}&amp;refresh={{$.ThisPath}}">
							{{if eq $LevelName (index $CurrentLogLevel 1)}}{{$.LightOpen}}{{end}}
							{{$LevelName}}
							{{if eq $LevelName (index $CurrentLogLevel 1)}}{{$.LightClose}}{{end}}
						</a>
					</td>
				{{end}}
			</tr>
		{{end}}
		</table>

		<h6>Miscellaneous</h6>
		<ul>
			<li><a href="/info" target="info">Application info [json]</a></li>
			<li><a href="/config" target="config">Prepared config [text]</a></li>
			<li>Profiler is
				<a href="/profiler-enable?refresh={{$.ThisPath}}">{{if .ProfilerEnabled}}{{$.LightOpen}}{{end}}ENABLED{{if .ProfilerEnabled}}{{$.LightClose}}{{end}}</a>
				<a href="/profiler-disable?refresh={{$.ThisPath}}">{{if not .ProfilerEnabled}}{{$.LightOpen}}{{end}}DISABLED{{if not .ProfilerEnabled}}{{$.LightClose}}{{end}}</a>
			</li>
			{{if .ProfilerEnabled}}
				<li><a href="debug/pprof/" target="pprof">Show profiler</a></li>
			{{end}}
			{{range .Extra}}
				<li>{{.}}</li>
			{{end}}
		</ul>

		<hr style="margin-top: 15px;" />
		<p class="top"><small><em>{{.Copyright}}</em></small></p>

	</body>
</html>
`

//----------------------------------------------------------------------------------------------------------------------------//