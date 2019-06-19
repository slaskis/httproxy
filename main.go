package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

func generateProxy(path, origin string, tls bool) http.Handler {
	return &httputil.ReverseProxy{Director: func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Host = origin
		req.URL.Host = origin
		if tls {
			req.URL.Scheme = "https"
		}
	}, Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	}}
}

type config struct {
	Path string
	Host string
}

func main() {
	var verbose bool
	var tls bool
	var addr string
	flag.StringVar(&addr, "addr", ":9001", "listening address")
	flag.BoolVar(&verbose, "verbose", false, "show logs")
	flag.BoolVar(&tls, "tls", false, "proxy to https")
	flag.Parse()

	// parse args "/sv=sv.example.com"
	var configuration []config
	for _, arg := range flag.Args() {
		parts := strings.Split(arg, "=")
		if len(parts) != 2 {
			log.Panicln("invalid argument")
		}
		configuration = append(configuration, config{
			Path: parts[0],
			Host: parts[1],
		})
	}

	if len(configuration) == 0 {
		log.Panicln("no configuration specified")
	}

	mux := http.NewServeMux()
	for _, conf := range configuration {
		if verbose {
			log.Printf("proxying %s => %s%s\n", conf.Path, conf.Host, conf.Path)
		}
		proxy := generateProxy(conf.Path, conf.Host, tls)
		mux.HandleFunc(conf.Path, func(w http.ResponseWriter, r *http.Request) {
			if verbose {
				log.Printf("%s %s => %s%s\n", r.Method, r.URL.String(), conf.Host, r.URL.String())
			}
			proxy.ServeHTTP(w, r)
		})
	}

	if verbose {
		log.Println("listening to " + addr)
	}
	log.Fatal(http.ListenAndServe(addr, mux))
}
