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

func generateProxy(cfg config) http.Handler {
	if cfg.Verbose {
		log.Printf("proxying %s => %s%s\n", cfg.SrcPath, cfg.Host, cfg.DstPath)
	}
	return &httputil.ReverseProxy{Director: func(req *http.Request) {
		if cfg.Verbose {
			log.Printf("%s %s => %s%s\n", req.Method, req.URL.String(), cfg.Host, cfg.DstPath)
		}
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Host = cfg.Host
		req.URL.Host = cfg.Host
		req.URL.Path = cfg.DstPath
		req.URL.Scheme = cfg.Scheme
	}, Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	}}
}

type config struct {
	SrcPath string
	DstPath string
	Host    string
	Scheme  string
	Verbose bool
}

func main() {
	var verbose bool
	var insecure bool
	var addr string
	flag.StringVar(&addr, "addr", ":9001", "listening address")
	flag.BoolVar(&verbose, "verbose", false, "show logs")
	flag.BoolVar(&insecure, "insecure", false, "proxy to http instead of https")
	flag.Parse()

	// parse args "/sv=sv.example.com" or "/x=:4000/"
	var configuration []config
	for _, arg := range flag.Args() {
		parts := strings.Split(arg, "=")
		if len(parts) != 2 {
			log.Panicln("invalid argument")
		}
		scheme := "https"
		if insecure {
			scheme = "http"
		}
		srcPath := parts[0]
		dstPath := parts[0]
		host := parts[1]
		parts = strings.Split(host, "/")
		if len(parts) == 2 {
			host = parts[0]
			dstPath = "/" + parts[1]
		}
		configuration = append(configuration, config{
			Verbose: verbose,
			Scheme:  scheme,
			SrcPath: srcPath,
			DstPath: dstPath,
			Host:    host,
		})
	}

	if len(configuration) == 0 {
		log.Panicln("no configuration specified")
	}

	mux := http.NewServeMux()
	for _, conf := range configuration {
		mux.Handle(conf.SrcPath, generateProxy(conf))
	}

	if verbose {
		log.Println("listening to " + addr)
	}
	log.Fatal(http.ListenAndServe(addr, mux))
}
