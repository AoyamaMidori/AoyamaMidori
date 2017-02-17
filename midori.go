package main

import (
	"bufio"
	"bytes"
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	// "io"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"
	"unicode"
	"unicode/utf8"

	"lib/some/irube/user"
)

const (
	_DEBUG = false

	fakeUserAgent = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36"
	mid           = "animation"
)

type token struct {
	Token    string `json:"token"`
	Nickname string `json:"nickname"`
}

func loop(tok *token, rawu *user.User, um *user.UserMid) {
	mode := float64(400 * time.Second)
	mean := float64(450 * time.Second)
	u, s := parameter(mode, mean)

	images, err := parseImages()
	if err != nil {
		log.Fatal(err)
	}

	master := template.New("master")
	master.Funcs(map[string]interface{}{
		"image": func() string {
			url := choice(images)
			return fmt.Sprintf(`<img src="%s" />`, url)
		},
	})

	if err := parseTemplate(master); err != nil {
		log.Fatal(err)
	}
	names := entryTemplates(master)

	buf := new(bytes.Buffer)
	buf.Grow(8192)

	for {
		buf.Reset()

		name := choice(names)
		err = master.ExecuteTemplate(buf, name, nil)
		if err != nil {
			log.Printf("template %q errored: %v", name, err)
			continue
		}

		brief, body := parseOutput(buf.Bytes())
		if brief == "" {
			log.Printf("template %q errored: you must specify the title", name)
			continue
		}

		var docid int64
		if !_DEBUG {
			docid, err = um.DW(brief, body)
			if err != nil {
				log.Printf("template %q errored: %v", name, err)
				continue
			}
		} else {
			log.Printf("Write(%v, %v)", brief, body)
		}

		d := time.Duration(logn(u, s))
		if !_DEBUG {
			log.Printf("http://www.ilbe.com/%v (%v)", docid, d)
			time.Sleep(d)
		} else {
			log.Printf("time.Sleep(d) where d = %v", d)
			time.Sleep(time.Second)
		}
	}
}

func parseOutput(output []byte) (brief string, body string) {
	var b []byte
	for len(output) > 0 {
		i := bytes.IndexRune(output, '\n')
		if i < 0 {
			i = len(output)
		}
		line := output[:i]
		output = output[i+1:]
		if len(line) == 0 {
			continue
		}
		if line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if bodySeparator(line) {
			break
		}
		line = bytes.TrimLeftFunc(line, unicode.IsSpace)
		if len(line) > 0 {
			b = append(b, line...)
		}
	}
	return string(b), string(output)
}

func pruneNewline(line []byte) []byte {
	if line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	return line
}

func bodySeparator(line []byte) bool {
	var repeat int
	for len(line) > 0 {
		r, size := utf8.DecodeRune(line)
		line = line[size:]
		if r != '=' {
			return false
		}
		repeat++
	}
	return repeat >= 3
}

func parseTemplate(master *template.Template) error {
	b, err := ioutil.ReadFile("template.tpl")
	if err != nil {
		return err
	}
	text := string(b)

	_, err = master.Parse(text)
	if err != nil {
		return err
	}
	return nil
}

func entryTemplates(master *template.Template) []string {
	const prefix = "entry-"
	templates := master.Templates()
	m := make(map[string]bool, len(templates)-1) // master template itself included
	for _, tpl := range templates {
		name := tpl.Name()
		// name "entry-" is not allowed.
		if !strings.HasPrefix(name, prefix) || len(name) <= len(prefix) {
			continue
		}
		if _, ok := m[name]; ok {
			log.Printf("template named %q has been parsed twice. The last one is applied", name)
		}
		m[name] = true
	}

	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	return names
}

func parseImages() ([]string, error) {
	r, err := os.Open("images")
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var urls []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		urlStr := scanner.Text()
		_, err = url.Parse(urlStr)
		if err != nil {
			break
		}
		urls = append(urls, urlStr)
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

func parameter(mode, mean float64) (float64, float64) {
	mode = math.Log(mode)
	mean = math.Log(mean)
	s := (mean - mode) / 3
	u := mean - s
	return u, math.Sqrt(2 * s)
}

func main() {
	tokens, err := getTokens()
	if err != nil {
		log.Fatal(err)
	}

	if len(tokens) == 0 {
		log.Fatal("No token available")
	}
	if len(tokens) > 1 {
		log.Fatal("Support only one token")
	}
	tok := tokens[0]
	// TODO: check tok.Token to be a valid token value

	if err = reseed(); err != nil {
		log.Fatal(err)
	}

	rawu := user.NewUser(tok.Token, fakeUserAgent)
	u := rawu.Mid(mid)

	loop(tok, rawu, u)
}

func reseed() error {
	bigSeed, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(1<<63-1))
	if err != nil {
		return err
	}
	rand.Seed(bigSeed.Int64())
	return nil
}

func getTokens() ([]*token, error) {
	r, err := os.Open("token.json")
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var s []*token
	err = json.NewDecoder(r).Decode(&s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func tick(d time.Duration) <-chan time.Time {
	tick := make(chan time.Time, 1)
	tick <- time.Now()
	go func() {
		for now := range time.Tick(d) {
			tick <- now
		}
	}()
	return tick
}

func logn(u, s float64) float64 {
	z := rand.NormFloat64()
	return math.Exp(u + s*z)
}

func choice(set []string) string {
	i := rand.Intn(len(set))
	return set[i]
}
