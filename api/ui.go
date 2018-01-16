package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"sync"

	"github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/labstack/echo"
)

var baseTemplate = `
{{define "base"}}
<!doctype html>
	<html>
	<head>
		<title>{{ .Title }}</title>
		<link rel="stylesheet" media="screen" href="https://govuk-elements.herokuapp.com/public/stylesheets/govuk-template.css?0.23.0">
		<link rel="stylesheet" media="screen" href="https://govuk-elements.herokuapp.com/public/stylesheets/govuk-elements-styles.css">
		<style>
			.table {
				display: table !important;
				width: 100%;
			}
			.tr {
				display: table-row !important;
			}
			.td {
				display: table-cell !important;
			}
			.org,.space, form {
				margin: 10px;
				padding: 10px;
				background: rgba(0,0,0,0.1);
			}
			table {
				width: 100%;
			}
			td.price {
				text-align: right;
				min-width: 160px;
			}
			td {
				padding: 5px !important;
			}
			strong {
				font-weight: 600 !important;
			}
		</style>
	</head>
	<body>
		<div class="phase-banner">
			<p>
				<strong class="phase-tag">ALPHA</strong>
				<span>This is a pre-release tool</span> – Don't rely on this data!
			</p>
		</div>
		<h1 class="heading-xlarge" style="margin:40px 10px">
			<span class="heading-secondary">GOV.UK PaaS</span>
			Usage Report
		</h1>
		{{ template "content" . }}
	</body>
	</html>
{{end}}

{{define "pricingtable" }}
	<div class="table">
		{{ range $row := . }}
			{{ if eq (0) (index $row "idx") }}
				<div class="tr">
					{{ range $key, $value := $row }}
						{{ if ne $key "idx" }}
							<div class="td">
								<label style="font-size:10px; display:block">{{ $key }}</label>
							</div>
						{{ end }}
					{{ end }}
				</div>
			{{ end }}
			<form method="POST" action="/pricing_plans/{{ index $row "id" }}" class="tr">
				{{ range $key, $value := $row }}
					{{ if ne $key "idx" }}
						<div class="td">
							<input name="{{ $key }}" placeholder="{{ $key }}" value="{{ $value }}">
						</div>
					{{ end }}
				{{ end }}
				<div class="td"><button class="button" type="submit" name="_method" value="PUT">Update</button></div>
				<div class="td"><button class="button" type="submit" name="_method" value="DELETE">Delete</button></div>
			</form>
		{{ end }}
	</div>
{{end}}
`

var templates = map[string]string{
	"default": `
		<p class="lede" class="margin:10px">
			{{ .Path }} data
		</p>
		<table>
			{{ range $row := .Rows }}
				{{ if eq (0) (index $row "idx") }}
					<tr>
						{{ range $key, $value := $row }}
							{{ if ne $key "idx" }}
								<th>{{ $key }}</th>
							{{ end }}
						{{ end }}
					</tr>
				{{ end }}
				<tr>
						
					{{ range $key, $value := $row }}
						{{ if ne $key "idx" }}
							<td>{{ $value }}</td>
						{{ end }}
					{{ end }}
				</tr>
			{{ end }}
		</table>
	`,
	"/pricing_plans": `
		<p class="lede" class="margin:10px">
			Pricing Plans
		</p>
		{{ template "pricingtable" .Rows }}
		<p class="lede" class="margin:10px">
			Create A new Plan
		</p>
		<div class="table">
			<form method="POST" action="/pricing_plans" class="tr">
				<div class="form-group">
					<label class="form-label">Plan name</label>
					<input class="form-control" name="name" placeholder="eg. Postgres Tiny" type="text" value="ComputePlanAlpha">
				</div>
				<div class="form-group">
					<label class="form-label">Valid From Date</label>
					<input class="form-control" name="valid_from" placeholder="2010-01-01T00:00:00Z" type="text" value="">
				</div>
				<div class="form-group">
					<label class="form-label">Service or Compute Plan GUID</label>
					<input class="form-control" name="plan_guid" placeholder="" type="text" value="f4d4b95a-f55e-4593-8d54-3364c25798c4">
				</div>
				<div class="form-group">
					<label class="form-label">
						Formula
						<span class="hint">You can use 
							<code>$time_in_seconds</code>,
							<code>$memory_in_mb</code>,
							<code>+</code>,
							<code>-</code>,
							<code>/</code>,
							<code>*</code>
						</span>
					</label>
					<input class="form-control" name="formula" placeholder="$time_in_seconds * 1.3" type="text" value="">
				</div>
				<button class="button" type="submit">Create pricing plan</button></div>
			</form>
			<form method="POST" action="/seed_pricing_plans" class="tr">
				<div class="td"><button class="button" type="submit">Seed missing pricing plans</button></div>
			</form>
		</div>
	`,
	"/pricing_plans/:pricing_plan_id": `
		<p class="lede" class="margin:10px">
			Pricing Plan Details
		</p>
		{{ template "pricingtable" .Rows }}
		<div>
			<a href="/pricing_plans">view all plans</a>
		</div>
	`,
	"/report/:org_guid": `
		{{ $from := .Range.From }}
		{{ $to := .Range.To }}

		<p class="lede" style="margin:20px 10px">
			Showing breakdown of resource usage between <strong>{{ $from }}</strong> to <strong>{{ $to }}</strong>
		</p>

		<form method="GET" action="/report" style="margin-bottom:30px;">
			<div style="padding: 20px; margin:2px; background:white; overflow:hidden">
				<div class="form-group" style="float:left; width: 25%; margin-right:40px;">
					<label class="form-label" for="rangeFrom">From date</label>
					<input class="form-control" id="rangeFrom" name="from" type="text" value="{{ $from }}" style="width:100%">
				</div>
				<div class="form-group" style="float:left; width:25%; margin-right:40px">
					<label class="form-label" for="rangeTo">To date</label>
					<input class="form-control" id="rangeTo" name="to" type="text" value="{{ $to }}" style="width:100%">
				</div>
				<div class="form-group" style="float:left;">
					<label class="form-label" for="rangeTo">&nbsp;</label>
					<input type="submit" class="button" value="Update range">
				</div>
			</div>
		</form>

		{{ range $org := .Rows }}
			<div class="org">
				<div class="space org-title">
					<table>
						<tr>
							<td><strong>Organisation {{ index $org "org_guid" | name }}</strong></td>
						</tr>
					</table>
				</div>
				{{ range $space := index $org "spaces" }}
					<div class="space">
						<table>
							<tr class="resource">
								<td colspan="4"><strong>Space {{ index $space "space_guid" | name }}</strong></td>
							</tr>
							{{ range $resource := index $space "resources" }}
								<tr class="resource">
									<td>{{ index $resource "name" }}</td>
									<td>{{ index $resource "pricing_plan_name" }}</td>
									<td class="price">{{ index $resource "price" | in_pounds}}</td>
								</tr>
							{{ end }}
							<tr class="resource">
								<td colspan="2"><strong>Space total</strong></td>
								<td class="price">{{ index $space "price" | in_pounds }}</td>
							</tr>
						</table>
					</div>
				{{ end }}
				<div class="space space-total">
					<table>
						<tr>
							<td><strong>Org total</strong></td>
							<td class="price">{{ index $org "price" | in_pounds}}</td>
						</tr>
					</table>
				</div>
			</div>
		{{ end }}
	`,
	"/repair": `
		<p class="lede" class="margin:10px">
			Repair Events
		</p>
		<div class="table">
			<p>
				This action will create missing START events for any apps or services that we have not seen events for but which are reported as running by cloud controller. It is like a diet version of the "purge and reset events" calls via the CC API.
			</p>
			<p>
				All events generated by this action have id=0 so you can remove any generated events by deleting all events with id=0
			</p>
			<form method="POST" action="/repair" class="tr">
				<div class="td"><button class="button" type="submit">Re generate missing events</button></div>
			</form>
		</div>
	
	`,
}

var (
	nameCacheMap  map[string]string
	nameCacheLock sync.Mutex
)

var templateFunctions = template.FuncMap{
	"in_pounds": func(pence float64) string {
		p := pence / 100.0
		return fmt.Sprintf("£ %.2f", p)
	},
	"name": func(guid string) (string, error) {
		nameCacheLock.Lock()
		defer nameCacheLock.Unlock()
		if nameCacheMap == nil {
			nameCacheMap = map[string]string{}
			cf, err := cloudfoundry.NewClient(&cfclient.Config{
				ApiAddress:        os.Getenv("CF_API_ADDRESS"),
				ClientID:          os.Getenv("CF_CLIENT_ID"),
				ClientSecret:      os.Getenv("CF_CLIENT_SECRET"),
				SkipSslValidation: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true",
			})
			if err != nil {
				return "", err
			}
			orgs, _ := cf.GetOrgs()
			for guid, org := range orgs {
				nameCacheMap[guid] = org.Name
			}
			spaces, _ := cf.GetSpaces()
			for guid, space := range spaces {
				nameCacheMap[guid] = space.Name
			}
		}
		name, ok := nameCacheMap[guid]
		if !ok {
			return guid, nil
		}
		return name, nil
	},
}

func compile(srcs map[string]string) map[string]*template.Template {
	tmpls := map[string]*template.Template{}
	for name, tmpl := range srcs {
		t := template.Must(template.New("base").Funcs(templateFunctions).Parse(baseTemplate))
		t = template.Must(t.Parse(` {{define "content"}} ` + tmpl + ` {{end}}`))
		tmpls[name] = t
	}
	return tmpls
}

var compiledTemplates = compile(templates)

// Render renders the JSON data as HTML
func Render(c echo.Context, r io.Reader, rt int) error {
	name := c.Path()
	tmpl, ok := compiledTemplates[name]
	if !ok {
		tmpl = compiledTemplates["default"]
	}
	return tmpl.Execute(c.Response(), &Data{
		Title: name,
		Path:  name,
		json:  r,
		rt:    rt,
		Range: c.Get("range").(RangeParams),
	})
}

type RowData map[string]interface{}

type Data struct {
	Title string
	Path  string
	json  io.Reader
	rt    int
	Range RangeParams
}

func (d *Data) Rows() chan RowData {
	ch := make(chan RowData, 30)
	go func() {
		defer close(ch)
		decode(d.json, ch, d.rt > 1)
	}()
	return ch
}

func decode(r io.Reader, rows chan RowData, many bool) {
	dec := json.NewDecoder(r)
	if many {
		if _, err := dec.Token(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	i := 0
	for dec.More() {
		var row RowData
		if err := dec.Decode(&row); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		row["idx"] = i
		rows <- row
		i++
	}
	if many {
		if _, err := dec.Token(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}
}
