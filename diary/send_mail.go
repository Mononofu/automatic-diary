package diary

import (
	"appengine"
	"appengine/datastore"
	"appengine/mail"
	"appengine/memcache"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func checkReminder(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		c.Errorf("Failed to load timezone")
		return
	}

	now := time.Now().In(loc)
	y, m, d := now.Date()
	cutoff := time.Date(y, m, d, 0, 0, 0, 0, loc)

	q := datastore.NewQuery("DiaryEntry").Filter("Date >", cutoff)
	t := q.Run(c)
	var e DiaryEntry
	_, err = t.Next(&e)

	if err == datastore.Done {
		// no entry yet for today - send reminder
		fmt.Fprintf(w, "Sending reminder email")
		sendReminder(c, cutoff)
	} else {
		fmt.Fprintf(w, "I already have an entry for today")
	}

}

func sendReminder(c appengine.Context, date time.Time) {
	addr := "j.schrittwieser@gmail.com"

	tag := fmt.Sprintf("diaryentry%dtag", rand.Int63())

	item := &memcache.Item{
		Key:   tag,
		Value: []byte(date.Format(time.RFC850)),
	}

	// Add the item to the memcache, if the key does not already exist
	if err := memcache.Add(c, item); err == memcache.ErrNotStored {
		c.Infof("item with key %q already exists", item.Key)
	} else if err != nil {
		c.Errorf("error adding item: %v", err)
	}

	msg := &mail.Message{
		Sender:  "Automatic Diary <diary@furidamu.org>",
		To:      []string{addr},
		Subject: "Entry reminder",
		Body:    fmt.Sprintf(reminderMessage, tag),
	}
	if err := mail.Send(c, msg); err != nil {
		c.Errorf("Couldn't send email: %v", err)
		return
	}
	c.Infof("Reminder mail sent for %v", date)
	c.Infof("body: %v", msg.Body)
}

const reminderMessage = `
Don't forget to update your diary!

Just respond to this message with todays entry.


-----
%v
`
