package diary

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	"bytes"
	"fmt"
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
	http.HandleFunc("/append", appendToEntry)
	http.HandleFunc("/append_submit", appendToEntrySubmit)

	// exposed for testing
	http.HandleFunc("/add_test_data", addTestData)
}

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

const entryTemplateHTML = `
<div class="entry">
    <h3>{{.Date.Format "Monday, 2. Jan"}}</h3>
    <p>{{.Content}}</p>
    <span><a href="/append?key={{.Key | urlquery }}">Append</a></span>
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

var baseTemplate = template.Must(template.New("body").Parse(baseTemplateHTML))
var entryTemplate = template.Must(template.New("entry").Parse(entryTemplateHTML))
var entryAppendTemplate = template.Must(template.New("entryAppend").Parse(entryAppendTemplateHTML))

type EntryContent struct {
	Date    time.Time
	Content string
	Key     string
}

type BodyContent struct {
	Body  string
	Title string
}

func showEntries(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	u := user.Current(c)
	if u == nil || !user.IsAdmin(c) {
		url, err := user.LoginURL(c, r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return
	}

	var doc bytes.Buffer

	q := datastore.NewQuery("DiaryEntry").Order("-Date")

	for t := q.Run(c); ; {
		var e DiaryEntry
		key, err := t.Next(&e)
		if err == datastore.Done {
			break
		}
		if err != nil {
			c.Errorf("failed to iterate over entries: %v", err)
			return
		}

		entryTemplate.Execute(&doc, EntryContent{
			Date:    e.Date,
			Content: strings.Replace(string(e.Content), "\n\n", "<br>", -1),
			Key:     key.Encode(),
		})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	baseTemplate.Execute(w, BodyContent{
		Body:  doc.String(),
		Title: "Home",
	})
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

func appendToEntry(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	args := r.URL.Query()
	rawKey := args.Get("key")
	key, err := datastore.DecodeKey(rawKey)

	if err != nil {
		c.Errorf("Failed to parse decode key '%v': %v", rawKey, err)
		return
	}

	var e DiaryEntry
	err = datastore.Get(c, key, &e)
	if err != nil {
		c.Errorf("failed to fetch entry: %v", err)
		return
	}

	var doc bytes.Buffer

	entryAppendTemplate.Execute(&doc, EntryContent{
		Date:    e.Date,
		Content: strings.Replace(string(e.Content), "\n\n", "<br>", -1),
		Key:     rawKey,
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	baseTemplate.Execute(w, BodyContent{
		Body:  doc.String(),
		Title: e.Date.Format("Monday, 2. Jan"),
	})

}

func appendToEntrySubmit(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	rawKey := r.FormValue("key")
	key, err := datastore.DecodeKey(rawKey)

	if err != nil {
		c.Errorf("Failed to parse decode key '%v': %v", rawKey, err)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		w.Header().Set("Status", "302")
		w.Header().Set("Location", "/append?key="+rawKey)
		return
	}

	var e DiaryEntry
	err = datastore.Get(c, key, &e)
	if err != nil {
		c.Errorf("failed to fetch entry: %v", err)
		return
	}

	newContent := fmt.Sprintf("%v\n\n\n\n<b>extended on %v</b>\n\n%v",
		string(e.Content),
		time.Now().Format("Monday, 2. Jan - 15:04"),
		content)

	e.Content = []byte(newContent)

	_, err = datastore.Put(c, key, &e)

	if err != nil {
		c.Errorf("failed to save entry: %v", err)
		return
	}

	w.Header().Set("Status", "302")
	w.Header().Set("Location", "/append?key="+rawKey)

}
