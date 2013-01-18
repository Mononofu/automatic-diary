package diary

import (
	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	"appengine/user"
	"bytes"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Attachment struct {
	Name         string
	Content      appengine.BlobKey
	Thumbnail    string
	BigImage     string
	ContentType  string
	CreationTime time.Time
}

type DiaryEntry struct {
	Author       string
	Content      []byte
	Date         time.Time
	CreationTime time.Time
	Attachments  []*datastore.Key
}

func init() {
	http.HandleFunc("/", showEntries)
	http.HandleFunc("/tasks/reminder", checkReminder)
	http.HandleFunc("/attachment", showAttachment)

	// append to existing entries
	http.HandleFunc("/append", appendToEntry)
	http.HandleFunc("/append_submit", appendToEntrySubmit)

	// exposed for testing
	http.HandleFunc("/add_test_data", addTestData)
	http.HandleFunc("/_ah/mail/", parseMail)
	http.HandleFunc("/test_parse", test_parse)
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

		var attachments bytes.Buffer

		for _, attachmentKey := range e.Attachments {

			if key != nil {
				var a Attachment
				err = datastore.Get(c, attachmentKey, &a)
				if err != nil {
					c.Errorf("failed to fetch entry for key '%v': %v", attachmentKey, err)
				} else {
					attachmentTemplate.Execute(&attachments, AttachmentContent{
						Name:      a.Name,
						Thumbnail: a.Thumbnail,
						Key:       string(a.Content),
					})
				}
			}
		}

		entryTemplate.Execute(&doc, EntryContent{
			Date:         e.Date,
			CreationTime: e.CreationTime,
			Content:      strings.Replace(string(e.Content), "\n", "<br>\n\n", -1),
			Key:          key.Encode(),
			Attachments:  attachments.String(),
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

func showAttachment(w http.ResponseWriter, r *http.Request) {
	args := r.URL.Query()
	rawKey := args.Get("key")
	blobstore.Send(w, appengine.BlobKey(rawKey))
}

func test_parse(w http.ResponseWriter, r *http.Request) {
	mail, err := parse_mail(mailtext)
	if err != nil {
		fmt.Fprintf(w, "failed to parse mail: %v", err)
	}

	keys := []string{}

	for key := range mail.Headers {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, name := range keys {
		fmt.Fprintf(w, "%v: %v\n\n", name, mail.Headers[name])
	}

	fmt.Fprintf(w, "\n\n\n--%v", mail.Plaintext)
}

const mailtext = `X-Received: by 10.42.39.1 with SMTP id f1mr7654524ice.4.1358539254163; 
        Fri, 18 Jan 2013 12:00:54 -0800 (PST) 
Return-Path: <j.schrittwieser@gmail.com> 
Received: from mail-ia0-x22d.google.com (mail-ia0-x22d.google.com [2607:f8b0:4001:c02::22d]) 
        by gmr-mx.google.com with ESMTPS id x4si728600igm.0.2013.01.18.12.00.54 
        (version=TLSv1 cipher=ECDHE-RSA-RC4-SHA bits=128/128); 
        Fri, 18 Jan 2013 12:00:54 -0800 (PST) 
Received-SPF: pass (google.com: domain of j.schrittwieser@gmail.com designates 2607:f8b0:4001:c02::22d as permitted sender) client-ip=2607:f8b0:4001:c02::22d; 
Authentication-Results: gmr-mx.google.com; 
       spf=pass (google.com: domain of j.schrittwieser@gmail.com designates 2607:f8b0:4001:c02::22d as permitted sender) smtp.mail=j.schrittwieser@gmail.com; 
       dkim=pass header.i=@gmail.com 
Received: by mail-ia0-f173.google.com with SMTP id l29so1770749iag.18 
        for <entry@automatic-diary.appspotmail.com>; Fri, 18 Jan 2013 12:00:54 -0800 (PST) 
DKIM-Signature: v=1; a=rsa-sha256; c=relaxed/relaxed; 
        d=gmail.com; s=20120113; 
        h=mime-version:x-received:date:message-id:subject:from:to 
         :content-type; 
        bh=cdafajGdyflKfsXC2aH8JUJUk5DqjD157g95dkUzlDI=; 
        b=BtBPCxlgvfJkIv94P4wtj956qfPb3UbhK/kYV2FVyPIbeZ/t5xJpKNpiQNeKObHJfP 
         gYA7gi2tCt6uCDfi7+gIy9K4M0Xf0KNPsjIVMk0wkkptntAkrth37IqG6CxINQWNWEL0 
         4fHZKaHWDPHvfNBSvKt3v4t2fk1dYw2JQigtI8VVf6/8t1WOq7nuwcmS9NMpLNjvSFve 
         oSv/BxkCcM6PyrJ93c7vKxKJ5rWaLaT/MgNSBY3h5iV3WXsXWx62nxaT7HrCXB01JUkU 
         MIiGNJ57rPB1vSjfbDTaphyFKu9ZN03zLzw/09ED8TMg6KcdzXSLfuTZnanpTy+iX1RC 
         Jf/w== 
MIME-Version: 1.0 
X-Received: by 10.50.151.166 with SMTP id ur6mr3118716igb.66.1358539254062; 
 Fri, 18 Jan 2013 12:00:54 -0800 (PST) 
Received: by 10.64.25.135 with HTTP; Fri, 18 Jan 2013 12:00:54 -0800 (PST) 
Date: Fri, 18 Jan 2013 21:00:54 +0100 
Message-ID: <CAHqcrsuQyApX=B9+g26D3m_TTC7o4EBW--ExDGN6gdP05GfshQ@mail.gmail.com> 
Subject: short test entry 
From: Julian Schrittwieser <j.schrittwieser@gmail.com> 
To: entry@automatic-diary.appspotmail.com 
Content-Type: multipart/mixed; boundary=e89a8f23549da6dffe04d395914e 
 
--e89a8f23549da6dffe04d395914e 
Content-Type: multipart/alternative; boundary=e89a8f23549da6dffb04d395914c 
 
--e89a8f23549da6dffb04d395914c 
Content-Type: text/plain; charset=UTF-8 
Content-Transfer-Encoding: quoted-printable 
 
Today, I spent a lot of time thinking about which watch and phone to buy, 
never being sure if it was actually worth it. Then I went and checked my 
accounts and realized that I only have about 4.5k right now and that this 
money needs to last till September. Sure, I get monthly payments from my 
parents and grandpa, but after rent and stuff only ~240=E2=82=AC are left o= 
ver. 
Which I'll probably spend on food alone, especially since SlowCarb / Paleo 
is expensive (but sure as hell worth it, I feel and look more ripped than 
ever) 
 
--e89a8f23549da6dffb04d395914c 
Content-Type: text/html; charset=UTF-8 
 
<div dir="ltr">blubeauo<div><br></div><div>aouaoeu</div></div> 
 
--e89a8f23549da6dffb04d395914c-- 
--e89a8f23549da6dffe04d395914e 
Content-Type: application/octet-stream; name="Diagram1.dia" 
Content-Disposition: attachment; filename="Diagram1.dia" 
Content-Transfer-Encoding: base64 
X-Attachment-Id: f_hc3r6vnu0 
 
H4sIAAAAAAACA72VTW/iMBCG7/wKy5ybQBEtVIRKe9hTj7vnapIMYVTHjuwJNJf97RuIWwo0UGjB 
UiJl9OZ5x+OPmTy+5kos0DoyOpL9oCcF6sSkpLNI/v3z+2YkH6edSUrwUD+ZhVzUf2i3+orknLl4 
CMPlchmoygEbGygqA4fhP1AKwloUymlHiI+AFBhWMR8FZktxySg05BjJGJKXzJpSp7JReV1ilLFi 
ASqS3dl6yNBjwi3OAXYBGcYW4aUd3avHeHwOukC7i80L46iWcFXsSVo4q/cHjVe5WqSzafcJmdF2 
m7R8cMP7LNlWI87BZqT3ver6qKYYt8FwMF6Nu/5oOOjd3g/e6nK6XXxdO3VdO3tdO3LPhbFsgXjf 
MjZGIejGlW2J5/u4BFS9xQ5Nq38+fUbM5kj+M1DuKxNowu8n7tTTm1lKDx/eLUULZUkpz59fL1Su 
hl5diL4gR7HCz7InzT+Gr34Gv7s66xt88/s3d0NDa20R6QiHODy9RWQlpeiObLNtTQtp7mXhsarv 
6r5amCa007DXFgoqtB7/a9OqhV9gf+MISJgWuLl/toDTzn8JGWOMeAgAAA== 
--e89a8f23549da6dffe04d395914e-- `
