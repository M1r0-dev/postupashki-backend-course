package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type responce struct {
	resp *http.Response
	body []byte
	err  error
	url  string
}

func printResp(resp *http.Response, body []byte) {
	fmt.Printf("%s %s\n", resp.Proto, resp.Status)

	for k, vs := range resp.Header {
		for _, v := range vs {
			fmt.Printf("%s: %s\n", k, v)
		}
	}
	fmt.Println()

	os.Stdout.Write(body)
}

func processReq(ctx context.Context, responces chan<- responce, u string) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		responces <- responce{
			err: err,
			url: u,
		}
		return
	}

	client := http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		responces <- responce{err: err, url: u}
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	responces <- responce{
		resp: resp,
		body: body,
		err:  err,
		url:  u,
	}
}

var (
	timeout = flag.Int("t", 15, "timeout in seconds")
	help    = flag.Bool("h", false, "show help")
)

func main() {
	flag.IntVar(timeout, "timeout", 15, "timeout in seconds")
	flag.BoolVar(help, "help", false, "show help")
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}
	urls := flag.Args()
	if len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "No URLs provided")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	responces := make(chan responce, len(urls))

	for _, u := range urls {
		go processReq(ctx, responces, u)
	}

	for i := 0; i < len(urls); i++ {
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "Error: timeout after %d seconds\n", *timeout)
			os.Exit(228)
		case res := <-responces:
			if res.err != nil {
				continue
			}
			cancel()
			printResp(res.resp, res.body)
			os.Exit(0)
		}
	}

	fmt.Fprintln(os.Stderr, "All requests failed")
	os.Exit(1)
}
