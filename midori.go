package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	// "io"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/fsnotify/fsnotify"
	"lib/some/irube/request"
	"lib/some/irube/user"
)

var mode = 200 * time.Second

var mean = 210 * time.Second

const (
	_DEBUG = false

	fakeUserAgent = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36"
	mid           = "animation"
)

type token struct {
	Token    string `json:"token"`
	Nickname string `json:"nickname"`
}

type fsEvent struct {
	fsnotify.Event
	Time time.Time
}

type fsEventFunc func(fsEvent)

type fsListener struct {
	fsw      *fsnotify.Watcher
	dispatch map[string]fsEventFunc
	alive    chan bool
}

func newFsListener() (*fsListener, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fsl := &fsListener{
		fsw:      fsw,
		dispatch: make(map[string]fsEventFunc),
		alive:    make(chan bool),
	}
	go fsl.fs()
	return fsl, nil
}

func (fsl *fsListener) fs() {
	fsw := fsl.fsw
	for {
		select {
		case ev := <-fsw.Events:
			callback := fsl.dispatch[ev.Name]
			if callback != nil {
				callback(fsEvent{
					Event: ev,
					Time:  time.Now(),
				})
			}
		case err := <-fsw.Errors:
			log.Print("fsListener on background: ", err)
		case <-fsl.alive:
			return
		}
	}
}

func (fsl *fsListener) Add(rel string, fn fsEventFunc) error {
	rel = filepath.Clean(rel)
	err := fsl.fsw.Add(rel)
	if err != nil {
		return err
	}
	fsl.dispatch[rel] = fn
	return nil
}

func (fsl *fsListener) Del(rel string) error {
	rel = filepath.Clean(rel)
	if _, ok := fsl.dispatch[rel]; !ok {
		return nil
	}
	delete(fsl.dispatch, rel)
	return fsl.fsw.Remove(rel)
}

func (fsl *fsListener) Close() error {
	defer func() { recover() }()
	close(fsl.alive)
	return fsl.fsw.Close()
}

type templateLoader struct {
	rawmaster *template.Template
	path      string
	master    *template.Template
	names     []string
	sync.RWMutex
}

func loadTemplate(master *template.Template, path string) (*templateLoader, error) {
	r := &templateLoader{
		rawmaster: master,
		path:      path,
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *templateLoader) load() error {
	r.Lock()
	defer r.Unlock()
	master, err := r.rawmaster.Clone()
	if err != nil {
		return err
	}
	err = parseTemplate(master, r.path)
	if err != nil {
		return err
	}

	r.master = master
	r.names = entryTemplates(master)
	return nil
}

func (r *templateLoader) template() (*template.Template, []string) {
	r.RLock()
	defer r.RUnlock()
	return r.master, r.names
}

func (r *templateLoader) fs(fsl *fsListener) error {
	return fsl.Add(r.path, r.fsFunc)
}

func (r *templateLoader) fsFunc(ev fsEvent) {
	switch ev.Op {
	case fsnotify.Create, fsnotify.Write:
		go r.load()
	}
}

type urlsLoader struct {
	path        string
	loadedTexts []string
	sync.RWMutex
}

func newURLsLoader(path string) (*urlsLoader, error) {
	r := &urlsLoader{
		path: path,
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *urlsLoader) load() error {
	r.Lock()
	defer r.Unlock()
	o, err := os.Open(r.path)
	if err != nil {
		return err
	}
	defer o.Close()

	scanner := bufio.NewScanner(o)

	var texts []string
	for scanner.Scan() {
		rawurl := scanner.Text()
		if rawurl == "" {
			continue
		}
		u, err := url.Parse(rawurl)
		if err != nil {
			return err
		}
		texts = append(texts, u.String())
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	r.loadedTexts = texts
	return nil
}

func (r *urlsLoader) texts() []string {
	r.RLock()
	defer r.RUnlock()
	return r.loadedTexts
}

func (r *urlsLoader) fs(fsl *fsListener) error {
	return fsl.Add(r.path, r.fsFunc)
}

func (r *urlsLoader) fsFunc(ev fsEvent) {
	switch ev.Op {
	case fsnotify.Create, fsnotify.Write:
		go r.load()
	}
}

func loop(tok *token, rawu *user.User, um *user.UserMid) {
	u, s := parameter(float64(mode), float64(mean))

	fsl, err := newFsListener()
	if err != nil {
		log.Printf("File system watcher is not supported on your platform %s: %v",
			runtime.GOOS, err)
	}

	imagesZip, err := zip.OpenReader("Aoyama Midori/images.zip")
	if err != nil {
		log.Fatal(err)
	}
	defer imagesZip.Close()

	bgmsZip, err := zip.OpenReader("Aoyama Midori/bgms.zip")
	if err != nil {
		log.Fatal(err)
	}
	defer bgmsZip.Close()

	// As executing a template may be concurrently run, lock is needed
	var addFileLock sync.Mutex
	master := template.New("master")
	master.Funcs(map[string]interface{}{
		"image": func() (string, error) {
			i := rand.Intn(len(imagesZip.File))
			o := imagesZip.File[i]

			r, err := o.Open()
			if err != nil {
				return "", err
			}
			defer r.Close()

			var url string
			if strings.HasSuffix(o.Name, ".txt") {
				b, err := ioutil.ReadAll(r)
				if err != nil {
					return "", err
				}
				url = strings.TrimSpace(string(b))
			} else {
				addFileLock.Lock()
				resp, err := um.AddFile(o.Name, r, request.WatermarkNone, -1)
				if err != nil {
					addFileLock.Unlock()
					return "", err
				}

				if len(resp.Files) == 0 {
					return "", errors.New("uploding file fail")
				}
				url = resp.Files[len(resp.Files)-1].URL
				addFileLock.Unlock()
			}

			return fmt.Sprintf(`<img src="%s" />`, url), nil
		},
		"bgm": func() (string, error) {
			i := rand.Intn(len(bgmsZip.File))
			o := bgmsZip.File[i]

			r, err := o.Open()
			if err != nil {
				return "", err
			}
			defer r.Close()

			var url string
			if strings.HasSuffix(o.Name, ".txt") {
				b, err := ioutil.ReadAll(r)
				if err != nil {
					return "", err
				}
				url = strings.TrimSpace(string(b))
			} else {
				addFileLock.Lock()
				resp, err := um.AddFile(o.Name, r, request.WatermarkNone, -1)
				if err != nil {
					addFileLock.Unlock()
					return "", err
				}

				if len(resp.Files) == 0 {
					return "", errors.New("uploding file fail")
				}
				url = resp.Files[len(resp.Files)-1].URL
				addFileLock.Unlock()
			}
			const format = `<embed src="%s" autostart="true" allowscriptaccess="always" enablehtmlaccess="true" allowfullscreen="true" width="422" height="240"></embed>`
			return fmt.Sprintf(format, url), nil
		},
	})

	tplLoader, err := loadTemplate(master, "Aoyama Midori/template.tpl")
	if err != nil {
		log.Fatal(err)
	}

	if fsl != nil {
		tplLoader.fs(fsl)
	}

	buf := new(bytes.Buffer)
	buf.Grow(8192)

	for {
		buf.Reset()

		master, names := tplLoader.template()
		name := choice(names)

		if err = um.VisitBoard(); err != nil {
			log.Printf("VisitBoard erroed: %v", err)
			continue
		}
		if _DEBUG {
			log.Print("VisitBoard()")
		}

		err = master.ExecuteTemplate(buf, name, struct {
			Nickname string
		}{
			Nickname: tok.Nickname,
		})
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

func parseTemplate(master *template.Template, name string) error {
	b, err := ioutil.ReadFile(name)
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

func parameter(mode, mean float64) (float64, float64) {
	mode = math.Log(mode)
	mean = math.Log(mean)
	s := (mean - mode) / 3
	u := mean - s
	return u, math.Sqrt(2 * s)
}

func parseArgs() (map[string]string, []string) {
	keywords := make(map[string]string)
	args := os.Args[1:]
	var keyword string
	for ; len(args) > 0; args = args[1:] {
		arg := args[0]
		if hasPrefixByte(arg, '-') {
			if keyword != "" {
				keywords[keyword] = ""
			}
			i := strings.IndexByte(arg, '=')
			if i >= 0 {
				keywords[arg[:i]] = arg[i+1:]
			} else {
				keyword = arg
			}
			continue
		}
		if keyword != "" {
			keywords[keyword] = arg
			keyword = ""
			continue
		}
		break
	}
	return keywords, args
}

func hasPrefixByte(s string, b byte) bool {
	if s == "" {
		return false
	}
	return s[0] == b
}

func main() {
	keywords, _ := parseArgs()
	if iskwdset(keywords, "-h", "-help", "--h", "--help") {
		fmt.Println("No help message is provided.")
		return
	}

	if arg, set := keywords["--mean"]; set {
		d, err := time.ParseDuration(arg)
		if err != nil {
			perror("--mean cannot be parsed", err)
			abort()
		}
		mean = d
	}

	if arg, set := keywords["--mode"]; set {
		d, err := time.ParseDuration(arg)
		if err != nil {
			perror("--mode cannot be parsed", err)
			abort()
		}
		mode = d
	}

	tokens, err := getTokens("Aoyama Midori/token.json")
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

func perror(s string, err error) { fmt.Fprintln(os.Stderr, s, ":", err) }

func abort() { os.Exit(1) }

func iskwdset(keywords map[string]string, kwds ...string) bool {
	for _, kwd := range kwds {
		if _, set := keywords[kwd]; set {
			return true
		}
	}
	return false
}

func reseed() error {
	bigSeed, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(1<<63-1))
	if err != nil {
		return err
	}
	rand.Seed(bigSeed.Int64())
	return nil
}

func getTokens(path string) ([]*token, error) {
	r, err := os.Open(path)
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
