# Presentation

`maildir-feed` is a daemon which fetchs RSS/Atom feeds and put them
directly into a Maildir mailbox.

# Setup

First, install the executables :

    go get github.com/sloonz/maildir-feed/maildir-feed

    go get github.com/sloonz/maildir-feed/maildir-feed-rss

This will download latest version, compile it, and put it in
`$GOPATH/bin`. You can then copy or link the two executables wherever you
want (`/usr/local/bin`, `$HOME/local/share`… Wherever, really, the only
restriction is that the two executables must be in the same directory).

Next, create the maildir in which the mails will be put :

    mkdir $HOME/Maildir-feeds

Create the cache directory :

    mkdir $HOME/.cache/rss2maildir

Then, create the configuration file:

    mkdir $HOME/.config/rss2maildir

    vim $HOME/.config/rss2maildir/feeds.json

# Usage

    /path/to/executables/maildir-feed

Daemon will not fork, you have to do it yourself or use a daemon manager
like `stop-start-daemon`.

# Configuration example

    {
    "IT": {
     "Slashdot": ["http://rss.slashdot.org/Slashdot/slashdotLinux",
                  "http://rss.slashdot.org/Slashdot/slashdotHardware"],
     "Ars Technica": "http://feeds.arstechnica.com/arstechnica/everything"
    },
    "Fun": {
     "SMBC": "http://feeds.feedburner.com/smbc-comics/PvLb",
     "XKCD": "http://xkcd.com/rss.xml"
    }
    }
