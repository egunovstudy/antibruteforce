package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"antibf/internal/model"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	baseURL := getenv("ANTIBF_URL", "http://localhost:8080")
	client := &http.Client{}

	var err error
	switch os.Args[1] {
	case "reset":
		err = resetCmd(client, baseURL, os.Args[2:])
	case "whitelist-add":
		err = addNetworkCmd(client, baseURL, "/api/v1/whitelist", os.Args[2:])
	case "whitelist-remove":
		err = removeNetworkCmd(client, baseURL, "/api/v1/whitelist", os.Args[2:])
	case "blacklist-add":
		err = addNetworkCmd(client, baseURL, "/api/v1/blacklist", os.Args[2:])
	case "blacklist-remove":
		err = removeNetworkCmd(client, baseURL, "/api/v1/blacklist", os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func resetCmd(client *http.Client, baseURL string, args []string) error {
	fs := flag.NewFlagSet("reset", flag.ContinueOnError)
	login := fs.String("login", "", "login")
	ip := fs.String("ip", "", "IPv4 address")
	if err := fs.Parse(args); err != nil {
		return err
	}

	payload := model.ResetRequest{Login: *login, IP: *ip}
	return doJSON(client, http.MethodPost, baseURL+"/api/v1/buckets/reset", payload)
}

func addNetworkCmd(client *http.Client, baseURL, endpoint string, args []string) error {
	fs := flag.NewFlagSet("network-add", flag.ContinueOnError)
	cidr := fs.String("cidr", "", "IPv4 CIDR")
	if err := fs.Parse(args); err != nil {
		return err
	}

	payload := map[string]string{"cidr": *cidr}
	return doJSON(client, http.MethodPost, baseURL+endpoint, payload)
}

func removeNetworkCmd(client *http.Client, baseURL, endpoint string, args []string) error {
	fs := flag.NewFlagSet("network-remove", flag.ContinueOnError)
	cidr := fs.String("cidr", "", "IPv4 CIDR")
	if err := fs.Parse(args); err != nil {
		return err
	}

	escaped := url.PathEscape(*cidr)
	return doJSON(client, http.MethodDelete, baseURL+endpoint+"/"+escaped, nil)
}

func doJSON(client *http.Client, method, target string, payload any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, target, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	fmt.Printf("success: %s\n", resp.Status)
	if len(respBody) > 0 {
		fmt.Println(string(respBody))
	}

	return nil
}

func getenv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func usage() {
	fmt.Println(`Usage:
  antibf reset --login alice --ip 192.168.1.10
  antibf whitelist-add --cidr 192.168.1.0/24
  antibf whitelist-remove --cidr 192.168.1.0/24
  antibf blacklist-add --cidr 10.0.0.0/8
  antibf blacklist-remove --cidr 10.0.0.0/8`)
}
