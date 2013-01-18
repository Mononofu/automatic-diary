package diary

import (
	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	"appengine/image"
	"appengine/memcache"
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type AttachmentJSON struct {
	Content     string
	ContentType string
	Name        string
}

func parseMail(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	defer r.Body.Close()

	var b bytes.Buffer
	if _, err := b.ReadFrom(r.Body); err != nil {
		c.Errorf("Error reading body: %v", err)
		return
	}

	mail := b.String()

	c.Debugf("Received mail: %v", mail)

	parsedMail, err := parse_mail(mail)
	if err != nil {
		c.Errorf("Failed while parsing mail: %v", err)
		return
	}

	rawBody := strings.Replace(parsedMail.Plaintext, "*", "", -1)

	cleanBody, err := getMailBody(rawBody)
	if err != nil {
		c.Errorf("error while parsing reply: %v", err)
		return
	}

	c.Infof("Received mail from %v: %v", parsedMail.Headers["From"], cleanBody)

	date, err := getReminderDate(c, rawBody)
	if err != nil {
		c.Errorf("error while parsing date: %v", err)
		return
	}

	attachments, err := storeAttachments(c, parsedMail.Attachments)
	if err != nil {
		c.Errorf("error while storing attachments: %v", err)
		return
	}

	e := DiaryEntry{
		Author:       "Julian",
		Content:      []byte(cleanBody),
		Date:         date,
		CreationTime: time.Now(),
		Attachments:  attachments,
	}

	_, err = datastore.Put(c, datastore.NewIncompleteKey(c, "DiaryEntry", nil), &e)
	if err != nil {
		c.Errorf("Failed to save to datastore: %s", err.Error())
		return
	}

	//c.Infof("received email: %v", body)
	c.Infof("at path: %v", r.URL.String())
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

	// strip of quote
	userText := reply[0:loc[0]]

	cleanText := ""

	lines := strings.Split(userText, "\n")
	for _, line := range lines {
		cleanText += line

		// arbitrary cutoff - gmail seems to wrap lines at around 75, but occasional
		// long words can push this to the left
		if len(line) < 65 { // line break made by the user - keep it!
			cleanText += "\n"
		} else { // replace newlines with spaces so words don't stick together
			cleanText += " "
		}
	}

	return strings.Trim(cleanText, " \n"), nil
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

func storeAttachments(c appengine.Context, rawAttachments []AttachmentJSON) ([]*datastore.Key, error) {
	keys := []*datastore.Key{}

	for _, rawAttachment := range rawAttachments {
		bytes, err := base64.StdEncoding.DecodeString(rawAttachment.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode attachment '%v':",
				rawAttachment.Name, err)
		}

		w, err := blobstore.Create(c, rawAttachment.ContentType)
		if err != nil {
			return nil, fmt.Errorf("failed to create blobstore entry: %v", err)
		}
		_, err = w.Write(bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to write to blobstore: %v", err)
		}
		err = w.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to close blobstore entry: %v", err)
		}

		blobKey, err := w.Key()
		if err != nil {
			return nil, fmt.Errorf("failed to get key for blobstore entry: %v", err)
		}

		thumbnailURL, err := image.ServingURL(c, blobKey, &image.ServingURLOptions{
			Secure: true,
			Size:   400,
			Crop:   false,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create thumbnail: %v", err)
		}

		bigImageURL, err := image.ServingURL(c, blobKey, &image.ServingURLOptions{
			Secure: true,
			Size:   1600,
			Crop:   false,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create big image: %v", err)
		}

		e := Attachment{
			Name:         rawAttachment.Name,
			Content:      blobKey,
			ContentType:  rawAttachment.ContentType,
			CreationTime: time.Now(),
			Thumbnail:    thumbnailURL.String(),
			BigImage:     bigImageURL.String(),
		}

		key, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Attachment", nil), &e)
		if err != nil {
			return nil, fmt.Errorf("Failed to save to datastore: %s", err)
		}

		keys = append(keys, key)
	}

	return keys, nil
}

func FindStringNthSubmatch(s string, regex string, n int) (string, error) {
	re, err := regexp.Compile(regex)
	if err != nil {
		return "", fmt.Errorf("Failed to compile regex: %v", err)
	}

	submatches := re.FindStringSubmatch(s)
	if n >= len(submatches) {
		return "", fmt.Errorf("not enough submatches for '%v' in: %v", regex, s)
	}
	return submatches[n], nil
}
