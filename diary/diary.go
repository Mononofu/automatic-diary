package diary

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	"bytes"
	"net/http"
	"strings"
	"time"
)

type DiaryEntry struct {
	Author       string
	Content      []byte
	Date         time.Time
	CreationTime time.Time
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
			Date:         e.Date,
			CreationTime: e.CreationTime,
			Content:      strings.Replace(string(e.Content), "\n", "<br>\n\n", -1),
			Key:          key.Encode(),
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
		Date:         (time.Now()).Add(time.Hour * 24),
		CreationTime: time.Now(),
	}

	_, _ = datastore.Put(c, datastore.NewIncompleteKey(c, "DiaryEntry", nil), &e)

	e = DiaryEntry{
		Author:       "Julian",
		Content:      []byte("It is a long established fact that a reader will be distracted by the readable content of a page when looking at its layout. The point of using Lorem Ipsum is that it has a more-or-less normal distribution of letters, as opposed to using 'Content here, content here', making it look like readable English. Many desktop publishing packages and web page editors now use Lorem Ipsum as their default model text, and a search for 'lorem ipsum' will uncover many web sites still in their infancy. Various versions have evolved over the years, sometimes by accident, sometimes on purpose (injected humour and the like)."),
		Date:         time.Now(),
		CreationTime: time.Now(),
	}

	_, _ = datastore.Put(c, datastore.NewIncompleteKey(c, "DiaryEntry", nil), &e)

	w.Header().Set("Status", "302")
	w.Header().Set("Location", "/")
}
