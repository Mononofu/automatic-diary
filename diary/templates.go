package diary

import (
	"text/template"
	"time"
)

const baseTemplateHTML = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>{{.Title}} &middot; Automatic Diary</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <!-- Le styles -->
    <link href="/assets/stylesheets/bootstrap.css" rel="stylesheet">
    <link href="/assets/stylesheets/main.css" rel="stylesheet">
    <script src="/assets/javascripts/jquery.js"></script>
  </head>

  <body>
    <div class="container-narrow">
        <div class="masthead">
        <ul class="nav nav-pills pull-right">
             <li class="active"><a href="/">Diary Entries</a></li>
             <li><a href="/tasks/reminder">Test Reminder</a></li>
             <li><a href="/add_test_data">Test Data</a></li>
             <li><a href="/_ah/admin/" target="_blank">Admin</a></li>
        </ul>
        <h3 class="muted">Automatic Diary</h3>
      </div>
      <hr>
        {{.Body}}
    </div>
  </body>
</html>
`

type BodyContent struct {
	Body  string
	Title string
}

const entryTemplateHTML = `
<div class="entry">
    <h3>{{.Date.Format "Monday, 2. Jan"}}</h3>
    <p>{{.Content}}</p>
    <span><i>Written on {{.CreationTime.Format "Monday, 2. Jan - 15:04"}}</i></span>
    <span class="append_link"><a href="/append?key={{.Key | urlquery }}">Append</a></span>
</div>
`

const entryAppendTemplateHTML = `
<div class="entry">
    <h3>{{.Date.Format "Monday, 2. Jan"}}</h3>
    <p>{{.Content}}</p>
    <form action="append_submit" method="post">
        <input type="hidden" name="key" value="{{.Key | urlquery }}">
        <textarea rows="5" name="content"></textarea>
        <button type="submit" class="btn btn-primary">Save changes</button>
        <button type="reset" class="btn">Reset</button>
    </form>
</div>
`

type EntryContent struct {
	Date         time.Time
	CreationTime time.Time
	Content      string
	Key          string
}

var baseTemplate = template.Must(template.New("body").Parse(baseTemplateHTML))
var entryTemplate = template.Must(template.New("entry").Parse(entryTemplateHTML))
var entryAppendTemplate = template.Must(template.New("entryAppend").Parse(entryAppendTemplateHTML))
