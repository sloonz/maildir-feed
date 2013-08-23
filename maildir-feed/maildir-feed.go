package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sloonz/go-maildir"
	"github.com/sloonz/go-mime-message"
	"github.com/sloonz/go-qprintable"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"
	"time"
)

const RUN_DELAY = 2
const INTERVAL_DELAY = 60 * 15
const MAX_ERR = 10

func Abs(p string) string {
	if path.IsAbs(p) {
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}
	return path.Join(wd, p)
}

func worker(root, md *maildir.Maildir, url_ *url.URL, delay int) {
	var execPath string

	errCount := 0
	dir, _ := path.Split(os.Args[0])
	if dir == "" {
		// maildir-feed in in $PTAH, assume that maildir-feed-rss is too
		execPath = "maildir-feed-rss"
	} else {
		execPath = path.Join(Abs(dir), "maildir-feed-rss")
	}

	time.Sleep(time.Duration(int64(delay) * 1e9))
	for {
		cmd := exec.Command(execPath, url_.String())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = nil
		cmd.Dir = md.Path
		err := cmd.Run()

		if err != nil {
			errCount++
			fmt.Fprintf(os.Stderr, "[%s: error: %s]\n", url_.String(), err.Error())
			if errCount >= MAX_ERR {
				errCount = 0

				msgText := fmt.Sprintf("Too many errors while fetching %s\n", url_.String())
				msg := message.NewTextMessage(qprintable.UnixTextEncoding, bytes.NewBufferString(msgText))
				msg.SetHeader("Date", time.Now().UTC().Format(time.RFC822))
				msg.SetHeader("From", "Feeds <feeds@localhost>")
				msg.SetHeader("To", "Feeds <feeds@localhost>")
				msg.SetHeader("Subject", "Error while fetching "+url_.String())
				msg.SetHeader("Content-Type", "text/plain; encoding=UTF-8")

				_, err = root.CreateMail(msg)
				if err != nil {
					panic(err.Error())
				}
			}
		}

		time.Sleep(time.Duration(INTERVAL_DELAY * 1e9))
	}
}

func parseBox(root, md *maildir.Maildir, data map[string]interface{}, delay int) int {
	for name, v := range data {
		child, err := md.Child(name, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't open %s: %s\n", name, err.Error())
			os.Exit(1)
		}

		switch v.(type) {
		case map[string]interface{}:
			delay = parseBox(root, child, v.(map[string]interface{}), delay)
		case []interface{}:
			for i, item := range v.([]interface{}) {
				url_, ok := item.(string)
				if !ok {
					fmt.Fprintf(os.Stderr, "%s[%i]: bad value type", name, i)
					os.Exit(1)
				}
				parsedURL, err := url.Parse(url_)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s[%i]: bad url: %s", name, i, err)
					os.Exit(1)
				}
				go worker(root, child, parsedURL, delay)
				delay += RUN_DELAY
			}
		case string:
			parsedURL, err := url.Parse(v.(string))
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: bad url: %s", name, err)
				os.Exit(1)
			}
			go worker(root, child, parsedURL, delay)
			delay += RUN_DELAY
		default:
			fmt.Fprintf(os.Stderr, "%s: bad value type", name)
			os.Exit(1)
		}
	}
	return delay
}

func main() {
	// Read & parse config
	configFile, err := os.Open(path.Join(os.Getenv("HOME"), ".config", "rss2maildir", "feeds.json"))
	config := make(map[string]interface{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't open config\n")
		return
	}
	data, err := ioutil.ReadAll(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't read config\n")
		return
	}
	if err = json.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %s\n", err.Error())
		return
	}

	// Open maildir
	var md_path string

	if len(os.Args) > 1 {
		md_path = Abs(os.Args[1])
	} else {
		md_path = path.Join(os.Getenv("HOME"), "Maildir-feeds")
	}

	md, err := maildir.New(md_path, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't open maildir: %s\n", err.Error())
		return
	}

	parseBox(md, md, config, 0)

	// Wait for SIGINT
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT)
	for {
		<-sigChan
		break
	}
}
