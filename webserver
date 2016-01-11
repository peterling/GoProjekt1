package main

import (
	"html/template"
	"io"
	"net/http"
	"os"
)

type EntetiesClass struct {
	Name  string
	Value int32
}

func start(w http.ResponseWriter, r *http.Request) {
	//	data, errr := readLines()
	//	if errr != nil {
	//		panic(errr)
	//	}

	data := map[string][]EntetiesClass{
		"Yoga":    {{"Yoga", 15}, {"Yoga", 51}},
		"Pilates": {{"Pilates", 3}, {"Pilates", 6}, {"Pilates", 9}},
	}
	t := template.New("t")
	t, err := t.Parse(htmlTemplate)
	if err != nil {
		panic(err)
	}

	err = t.Execute(os.Stdout, data)
	err = t.ExecuteTemplate(w, "root", data)
	if err != nil {
		panic(err)
	}
}

var mux map[string]func(http.ResponseWriter, *http.Request)

func main() {
	server := http.Server{
		Addr:    ":8000",
		Handler: &myHandler{},
	}

	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	mux["/"] = start

	server.ListenAndServe()
}

type myHandler struct{}

func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := mux[r.URL.String()]; ok {
		h(w, r)
		return
	}
	io.WriteString(w, "My server: "+r.URL.String())
}

const rootHTML = `
{{define "root"}}
<html>
	<head>
		<title>{{.Title}} - runsit</title>
		<style>
		.output {
		   font-family: monospace;
		   font-size: 10pt;
		   border: 2px solid gray;
		   padding: 0.5em;
		   overflow: scroll;
		   max-height: 25em;
		}
		.output div.stderr {
		   color: #c00;
		}
		.output div.system {
		   color: #00c;
		}
                .topbar {
                    font-family: sans;
                    font-size: 10pt;
                }
		</style>
	</head>
	<body>
                {{if .RootLink}}
                    <div id='topbar'>runsit on <a href="{{.RootLink}}">{{.Hostname}}</a>.
                {{end}}
		<h1>{{.Title}}</h1>
		{{template "body" .}}
	</body>
</html>
{{end}}
`

var htmlTemplate = `{{define "root"}}
<html>
	<head>
		<title>Test</title>
		<style>
		.output {
		   font-family: monospace;
		   font-size: 10pt;
		   border: 2px solid gray;
		   padding: 0.5em;
		   overflow: scroll;
		   max-height: 25em;
		}
		.output div.stderr {
		   color: #c00;
		}
		.output div.system {
		   color: #00c;
		}
                .topbar {
                    font-family: sans;
                    font-size: 10pt;
                }
		</style>
	</head>
	<body>
      {{range $index, $element := .}}{{$index}}
{{range $element}}{{.Value}}
{{end}}
{{end}}
	</body>
</html>
{{end}}`

var templateHTML = map[string]string{
	"taskList": `
	{{define "body"}}
		<h2>Running</h2>
		<ul>
		{{range $index, $element := .}}{{$index}}
		<li><a href='/task/{{range $element}}'>{{range $element}}</a>:</li>
		{{end}}			
		{{end}}
		</ul>
		<h2>Log</h2>
		<pre>{{.Log}}</pre>
	{{end}}
`,
}
