package diary

import (
	"appengine"
	"appengine/datastore"
	"bytes"
	"net/http"
	"strings"
	"text/template"
	"time"
)

type DiaryEntry struct {
	Author  string
	Content []byte
	Date    time.Time
}

func init() {
	http.HandleFunc("/", showEntries)
	http.HandleFunc("/incoming_mail", incomingMail)
	http.HandleFunc("/tasks/reminder", checkReminder)

	// exposed for testing
	http.HandleFunc("/add_test_data", addTestData)
}

const baseTemplateHTML = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>Home &middot; Automatic Diary</title>
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

        {{.}}

    </div>
  </body>
</html>
`

const entryTemplateHTML = `
<div class="entry">
    <h3>{{.Date.Format "Monday, 2. Jan"}}</h3>
    <p>{{.Content}}</p>
</div>
`

var baseTemplate = template.Must(template.New("body").Parse(baseTemplateHTML))
var entryTemplate = template.Must(template.New("entry").Parse(entryTemplateHTML))

type EntryContent struct {
	Date    time.Time
	Content string
}

func showEntries(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	q := datastore.NewQuery("DiaryEntry").Order("-Date")

	var doc bytes.Buffer
	for t := q.Run(c); ; {
		var e DiaryEntry
		_, err := t.Next(&e)
		if err == datastore.Done {
			break
		}
		if err != nil {
			c.Errorf("failed to iterate over entries: %v", err)
			return
		}
		entryTemplate.Execute(&doc, EntryContent{e.Date,
			strings.Replace(string(e.Content), "\n\n", "<br>", -1)})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	baseTemplate.Execute(w, doc.String())
}

func addTestData(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	e := DiaryEntry{
		Author: "Julian",
		Content: []byte(`Lorem Ipsum is simply dummy text of the printing and typesetting industry. 

            Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.`),
		Date: (time.Now()).Add(time.Hour * 24),
	}

	_, _ = datastore.Put(c, datastore.NewIncompleteKey(c, "DiaryEntry", nil), &e)

	e = DiaryEntry{
		Author:  "Julian",
		Content: []byte("It is a long established fact that a reader will be distracted by the readable content of a page when looking at its layout. The point of using Lorem Ipsum is that it has a more-or-less normal distribution of letters, as opposed to using 'Content here, content here', making it look like readable English. Many desktop publishing packages and web page editors now use Lorem Ipsum as their default model text, and a search for 'lorem ipsum' will uncover many web sites still in their infancy. Various versions have evolved over the years, sometimes by accident, sometimes on purpose (injected humour and the like)."),
		Date:    time.Now(),
	}

	_, _ = datastore.Put(c, datastore.NewIncompleteKey(c, "DiaryEntry", nil), &e)

	w.Header().Set("Status", "302")
	w.Header().Set("Location", "/")
}
