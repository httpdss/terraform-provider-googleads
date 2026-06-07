package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const scope = "https://www.googleapis.com/auth/adwords"

func main() {
	credentialsFile := flag.String("credentials", "credentials.json", "OAuth client credentials JSON downloaded from Google Cloud Console")
	outFile := flag.String("out", "token.json", "Path to save OAuth token JSON")
	printEnv := flag.Bool("print-env", true, "Print Terraform provider environment variable exports after saving the token")
	flag.Parse()

	b, err := os.ReadFile(*credentialsFile)
	if err != nil {
		log.Fatalf("read credentials file: %v", err)
	}
	cfg, err := google.ConfigFromJSON(b, scope)
	if err != nil {
		log.Fatalf("parse credentials file: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("start local callback listener: %v", err)
	}
	defer listener.Close()
	cfg.RedirectURL = "http://" + listener.Addr().String() + "/oauth2callback"

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		if errText := r.URL.Query().Get("error"); errText != "" {
			errCh <- fmt.Errorf("oauth error: %s", errText)
			http.Error(w, errText, http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("callback did not include code")
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(w, "Google Ads token received. You can close this browser tab.")
		codeCh <- code
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())

	url := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Println("Open this URL in a browser and authorize Google Ads access:")
	fmt.Println(url)
	fmt.Println("\nWaiting for the browser callback on", cfg.RedirectURL)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		log.Fatal(err)
	case <-time.After(5 * time.Minute):
		log.Fatal("timed out waiting for OAuth callback")
	}

	tok, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("exchange authorization code: %v", err)
	}
	if tok.RefreshToken == "" {
		log.Fatal("no refresh token returned; revoke the app grant and retry, or force consent with a new OAuth client")
	}
	out, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		log.Fatalf("marshal token: %v", err)
	}
	if err := os.WriteFile(*outFile, out, 0600); err != nil {
		log.Fatalf("write token file: %v", err)
	}
	fmt.Printf("\nSaved token to %s\n", *outFile)
	if *printEnv {
		fmt.Println("\nTerraform provider environment variables:")
		fmt.Printf("export GOOGLEADS_CREDENTIALS_FILE=%q\n", *credentialsFile)
		fmt.Printf("export GOOGLEADS_TOKEN_FILE=%q\n", *outFile)
		fmt.Println("export GOOGLEADS_DEVELOPER_TOKEN=\"your-developer-token\"")
		fmt.Println("export GOOGLEADS_CUSTOMER_ID=\"1234567890\"")
		fmt.Println("# Optional for manager account access:")
		fmt.Println("export GOOGLEADS_LOGIN_CUSTOMER_ID=\"0987654321\"")
	}
}
