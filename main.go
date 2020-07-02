package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func generateProxy(cfg config) http.Handler {
	if cfg.Verbose {
		log.Printf("proxying %s => %s%s\n", cfg.SrcPath, cfg.Host, cfg.DstPath)
	}
	return &httputil.ReverseProxy{Director: func(req *http.Request) {
		oldPath := req.URL.String()
		newPath := cfg.DstPath + strings.TrimPrefix(oldPath, cfg.SrcPath)
		if cfg.Verbose {
			log.Printf("%s %s => %s%s\n", req.Method, oldPath, cfg.Host, newPath)
		}
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Host = cfg.Host
		req.URL.Host = cfg.Host
		req.URL.Path = newPath
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

	// parse args "/sv=sv.example.com" or "/x=:4000/" or "--" to start parsing a command
	var configuration []config
	var command []string
	state := "config"
	for _, arg := range flag.Args() {
		if arg == "--" {
			state = "command"
			continue
		}

		switch state {
		case "config":
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

		case "command":
			command = append(command, arg)
		}
	}

	if len(configuration) == 0 {
		log.Panicln("no configuration specified")
	}

	mux := http.NewServeMux()
	for _, conf := range configuration {
		mux.Handle(conf.SrcPath, generateProxy(conf))
	}

	srv := http.Server{
		Handler: mux,
		Addr:    addr,
	}

	var exitCode int

	if len(command) > 0 {
		// run the command and shutdown proxy if command stops
		go func() {
			cmd := exec.Command(command[0], command[1:]...)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Env = os.Environ()
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				log.Println("error from command", err)
			}
			exitCode = cmd.ProcessState.ExitCode()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = srv.Shutdown(ctx)
			if err != nil {
				log.Println("error white shutting down http server", err)
			}
		}()
	}

	if verbose {
		log.Println("listening to " + addr)
	}
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Println(err)
	}

	os.Exit(exitCode)
}
