package diary

import (
	"appengine"
	"appengine/datastore"
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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
		Content: strings.Replace(string(e.Content), "\n", "<br>\n\n", -1),
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
