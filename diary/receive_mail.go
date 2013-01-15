package diary

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type MailJSON struct {
	TextBody string
	From     string
}

func incomingMail(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	defer r.Body.Close()
	var b bytes.Buffer
	if _, err := b.ReadFrom(r.Body); err != nil {
		c.Errorf("Error reading body: %v", err)
		return
	}

	var m MailJSON

	err := json.Unmarshal(b.Bytes(), &m)

	if err != nil {
		c.Errorf("failed to decode mail")
		return
	}

	rawBody := strings.Replace(m.TextBody, "*", "", -1)

	body, err := getMailBody(rawBody)
	if err != nil {
		c.Errorf("error while parsing reply: %v", err)
		return
	}

	date, err := getReminderDate(c, rawBody)
	if err != nil {
		c.Errorf("error while parsing date: %v", err)
		return
	}

	c.Infof("Received mail from %s: %s", m.From, body)
	e := DiaryEntry{
		Author:  "Julian",
		Content: []byte(body),
		Date:    date,
	}

	_, err = datastore.Put(c, datastore.NewIncompleteKey(c, "DiaryEntry", nil), &e)
	if err != nil {
		c.Errorf("Failed to save to datastore: %s", err.Error())
		return
	}
}

func getMailBody(reply string) (string, error) {
	re, err := regexp.Compile(".*On.*(\\n)?wrote:")
	if err != nil {
		return "", fmt.Errorf("Failed to compile regex: %v", err)
	}

	loc := re.FindStringIndex(reply)

	if loc == nil {
		return "", fmt.Errorf("Failed to parse reply")
	}

	return reply[0:loc[0]], nil
}

func getReminderDate(c appengine.Context, text string) (time.Time, error) {
	re, err := regexp.Compile(`diaryentry\d+tag`)
	if err != nil {
		return time.Now(), fmt.Errorf("Failed to compile regex: %v", err)
	}

	tag := re.FindString(text)

	if tag == "" {
		return time.Now(), fmt.Errorf("Failed to match tag")
	}

	item, err := memcache.Get(c, tag)
	if err == memcache.ErrCacheMiss {
		return time.Now(), fmt.Errorf("item not in the cache")
	} else if err != nil {
		return time.Now(), fmt.Errorf("error getting item: %v", err)
	}

	date, err := time.Parse(time.RFC850, string(item.Value))
	if err != nil {
		return time.Now(), fmt.Errorf("failed to parse date: %v", err)
	}

	return date, nil
}
