package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
)

type Sender struct {
	Hostname string
}

func Send(from string, to string, data []byte) error {
	log.Printf("====== Start Send from %s to %s ======\n", from, to)
	addr := to

	_, mxdomain, err := splitAddress(addr)
	if err != nil {
		return err
	}

	mxservers, err := net.LookupMX(mxdomain)
	if err != nil {
		return err
	}
	if len(mxservers) == 0 {
		mxservers = []*net.MX{{Host: mxdomain}}
	}

	for _, mxserver := range mxservers {
		c, err := smtp.Dial(mxserver.Host + ":25")
		log.Println("Host: ", mxserver.Host)
		if err != nil {
			return err
		}

		if err := c.Hello(domain); err != nil {
			return err
		}

		if ok, _ := c.Extension("STARTTLS"); ok {
			log.Println("StartTLS!")
			tlsConfig := &tls.Config{
				ServerName:         mxserver.Host,
				InsecureSkipVerify: true,
			}
			if err := c.StartTLS(tlsConfig); err != nil {
				log.Println("StartTLS err: ", err.Error())
				return err
			}
		}

		opt := &smtp.MailOptions{
			Size: len(data),
		}

		if err := c.Mail(from, opt); err != nil {
			return err
		}

		if err := c.Rcpt(addr); err != nil {
			return err
		}

		wc, err := c.Data()
		if err != nil {
			return err
		}

		if _, err := wc.Write(data); err != nil {
			return err
		}
		if err := wc.Close(); err != nil {
			return err
		}

		if err := c.Quit(); err != nil {
			return err
		} else {
			return nil
		}
	}

	return nil
}

// The SMTP server Backend
type Backend struct{}

func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return &Session{}, nil
}

func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return nil, smtp.ErrAuthRequired
}

// A Session is returned after successful login.
type Session struct {
	from string
	to   string
}

func (s *Session) Mail(from string, opts smtp.MailOptions) error {
	log.Println("Mail from:", from)
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string) error {
	log.Println("Rcpt to:", to)
	s.to = to
	return nil
}

func (s *Session) Data(r io.Reader) error {
	var data []byte
	var err error
	if data, err = ioutil.ReadAll(r); err != nil {
		return err
	}
	err = Send(s.from, s.to, data)
	if err != nil {
		log.Printf("^^^^^^ Send from %s to %s failed %v ^^^^^^\n", s.from, s.to, err)
	} else {
		log.Printf("^^^^^^ Send from %s to %s succeed! ^^^^^^\n", s.from, s.to)
	}
	return err
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

func splitAddress(addr string) (local, domain string, err error) {
	parts := strings.SplitN(addr, "@", 2)
	if len(parts) != 2 {
		return "", "", errors.New("mta: invalid mail address")
	}
	return parts[0], parts[1], nil
}

var (
	domain string
	port   int
	help   bool
)

func init() {
	flag.StringVar(&domain, "domain", "least-mta", "MTA server domain, default: least-mta.")
	flag.IntVar(&port, "port", 25, "MTA server port, default: 25.")
	flag.BoolVar(&help, "help", false, "Show help")
}

func main() {
	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	be := &Backend{}

	s := smtp.NewServer(be)

	s.Addr = fmt.Sprintf(":%d", port)
	s.Domain = domain
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true
	s.Debug = os.Stdout
	log.SetOutput(os.Stdout)

	log.Printf("Starting server at %s%s", s.Domain, s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
