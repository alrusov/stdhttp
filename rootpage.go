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
		<h4><em>{{.Name}} [{{.AppName}} {{.AppVersion}}{{if .AppTags}}&nbsp;{{.AppTags}}{{end}}]</em></h4>

		<h6>Logging level</h6>
		<table class="grd">
		{{range $_, $CurrentLogLevel := .LogLevels}}
			<tr>
				<th class="left">
					{{if index $CurrentLogLevel 0}}{{index $CurrentLogLevel 0}}{{else}}main{{end}}
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
