package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sloonz/cfeedparser"
	"github.com/sloonz/go-maildir"
	"github.com/sloonz/go-mime-message"
	"github.com/sloonz/go-qprintable"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

type Cache struct {
	data map[string]bool
	path string
}

func (c *Cache) load() error {
	cacheFile, err := os.Open(c.path)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(cacheFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &c.data)
}

func (c *Cache) dump() error {
	cacheFile, err := os.Create(c.path + ".new")
	if err != nil {
		return err
	}
	defer cacheFile.Close()

	enc := json.NewEncoder(cacheFile)
	if err = enc.Encode(c.data); err != nil {
		return err
	}

	return os.Rename(c.path+".new", c.path)
}

var cache Cache

func firstNonEmpty(s ...string) string {
	var val string
	for _, val = range s {
		if val != "" {
			break
		}
	}
	return val
}

func getRFC822Date(e *feedparser.Entry) string {
	emptyTime := time.Time{}
	if e.PublicationDateParsed != emptyTime {
		return e.PublicationDateParsed.Format(time.RFC1123Z)
	}
	if e.ModificationDateParsed != emptyTime {
		return e.ModificationDateParsed.Format(time.RFC1123Z)
	}
	if e.PublicationDate != "" {
		return e.PublicationDate
	}
	if e.ModificationDate != "" {
		return e.ModificationDate
	}
	return time.Now().UTC().Format(time.RFC1123Z)
}

func getFrom(e *feedparser.Entry) string {
	name := strings.TrimSpace(message.EncodeWord(firstNonEmpty(e.Author.Name, e.Author.Uri, e.Author.Text)))
	if e.Author.Email != "" {
		name += " <" + strings.TrimSpace(e.Author.Email) + ">"
	} else {
		name += " <noreply@localhost>"
	}
	return name
}

var convertEOLReg = regexp.MustCompile("\r\n?")

func convertEOL(s string) string {
	return convertEOLReg.ReplaceAllString(s, "\n")
}

func process(rawUrl string) error {
	url_, err := url.Parse(rawUrl)
	if err != nil {
		return err
	}

	md, err := maildir.New(".", false)
	if err != nil {
		return err
	}

	feed, err := feedparser.ParseURL(url_)
	if err != nil {
		return err
	}

	fmt.Printf("[%s]\n", feed.Title)
	for _, entry := range feed.Entries {
		postId := firstNonEmpty(entry.Id, entry.Link, entry.PublicationDate+":"+entry.Title)
		if _, hasId := cache.data[postId]; hasId {
			continue
		}

		body := convertEOL(firstNonEmpty(entry.Content, entry.Summary))
		body += "\n<p><small><a href=\"" + entry.Link + "\">View post</a></small></p>\n"

		title := strings.TrimSpace(entry.Title)
		msg := message.NewTextMessage(qprintable.UnixTextEncoding, bytes.NewBufferString(body))
		msg.SetHeader("Date", getRFC822Date(&entry))
		msg.SetHeader("From", getFrom(&entry))
		msg.SetHeader("To", "Feeds <feeds@localhost>")
		msg.SetHeader("Subject", message.EncodeWord(title))
		msg.SetHeader("Content-Type", "text/html; charset=\"UTF-8\"")

		_, err = md.CreateMail(msg)
		if err != nil {
			return err
		}

		fmt.Printf("  %s\n", title)
		cache.data[postId] = true
	}

	return nil
}

func main() {
	url_ := os.Args[1]

	cache.path = path.Join(os.Getenv("HOME"), ".cache", "rss2maildir", strings.Replace(url_, "/", "_", -1))
	cache.data = make(map[string]bool)

	err := cache.load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: can't read cache: %s\n", err.Error())
	}

	err = process(url_)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't process feed: %s\n", err.Error())
		os.Exit(1)
	}

	err = cache.dump()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't write cache: %s\n", err.Error())
		os.Exit(1)
	}
}
