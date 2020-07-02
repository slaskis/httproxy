package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
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
	flag.BoolVar(&insecure, "insecure", false, "default to http scheme")
	flag.Parse()

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
			cfg, err := parseArg(arg, verbose, insecure)
			if err != nil {
				log.Panicln(err)
			}
			configuration = append(configuration, cfg)
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
			if verbose {
				log.Println("starting command:", command)
			}
			cmd := exec.Command(command[0], command[1:]...)
			prepare(cmd)
			cmd.Env = os.Environ()
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				log.Println("error from command", err)
			}
			exitCode = cmd.ProcessState.ExitCode()
			if verbose {
				log.Println("command exited:", exitCode)
			}

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

func parseArg(arg string, verbose, insecure bool) (config, error) {
	parts := strings.Split(arg, "=")
	if len(parts) != 2 {
		return config{}, fmt.Errorf("invalid argument: %s", arg)
	}
	source := parts[0]
	target := parts[1]
	scheme := "https"
	if insecure {
		scheme = "http"
	}
	if !strings.Contains(target, "://") {
		target = scheme + "://" + target
	}
	srcPath := source
	dstPath := source
	u, err := url.Parse(target)
	if err != nil {
		return config{}, err
	}
	if u.Path != "" {
		dstPath = u.Path
	}
	return config{
		Verbose: verbose,
		Scheme:  u.Scheme,
		SrcPath: srcPath,
		DstPath: dstPath,
		Host:    u.Host,
	}, nil
}
