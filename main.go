package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
)

const gfwList = `https://github.com/gfwlist/gfwlist/raw/master/gfwlist.txt`

var list []string

func readList(data []byte) {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(dst, data)
	if err == nil {
		data = dst[:n]
	}

	if !bytes.HasPrefix(data, []byte("[AutoProxy ")) {
		log.Fatal("invalid auto proxy file")
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.ContainsRune(line, '.') {
			continue
		}
		domain := line
		switch line[0] {
		case '.':
			domain = "|h://" + line[1:]
			fallthrough
		case '|':
			if line[1] == '|' {
				domain = strings.TrimRight(domain[2:], "/")
				if strings.ContainsRune(domain, '/') {
					log.Printf("unsupported line: %s", line)
					continue
				}
			} else {
				u, err := url.Parse(strings.Replace(domain[1:], "*", "/", -1))
				if err != nil || !strings.ContainsRune(u.Host, '.') || strings.ContainsRune(u.Host, ':') {
					log.Printf("unsupported line: %s", line)
					continue
				}
				domain = u.Host
			}
			if net.ParseIP(domain) == nil {
				list = append(list, domain)
			}
		}
	}
}

func readOnline() {
	resp, err := http.Get(gfwList)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("online request returned code: %d", resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	readList(data)
}

func readFile(file string) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	readList(data)
}

func main() {
	if len(os.Args) > 1 {
		readFile(os.Args[1])
	} else {
		readOnline()
	}
	sort.Strings(list)

	fil, err := os.Create("domain.conf")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := fil.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	for i := range list {
		if i != 0 && list[i] == list[i - 1] {
			continue
		}
		_, err := fil.WriteString(list[i] + "\n")
		if err != nil {
			log.Fatal(err)
		}
	}
}
