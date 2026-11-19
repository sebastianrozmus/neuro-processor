// Command neuro-share is the server that collects the conversations
// players agreed to share. See internal/share for the protocol and
// deploy/share for running it in Docker.
//
//	neuro-share            serve (settings from SHARE_* variables)
//	neuro-share verify     check every stored signature and hash chain
//	neuro-share health     exit 0 when the local server answers
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/shareserver"
)

func main() {
	data := env("SHARE_DATA", "/data")

	if len(os.Args) > 1 && os.Args[1] == "verify" {
		os.Exit(verify(data))
	}

	// health is the container's health check: the image has no curl.
	if len(os.Args) > 1 && os.Args[1] == "health" {
		client := &http.Client{Timeout: 3 * time.Second}

		addr := env("SHARE_ADDR", ":8080")
		if strings.HasPrefix(addr, ":") {
			addr = "127.0.0.1" + addr
		}

		response, err := client.Get("http://" + addr + "/healthz")
		if err != nil || response.StatusCode != http.StatusOK {
			os.Exit(1)
		}

		os.Exit(0)
	}

	if len(os.Args) > 1 {
		fmt.Fprintln(os.Stderr, "użycie: neuro-share [verify]")
		os.Exit(2)
	}

	if err := serve(data); err != nil {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func number(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, ""))
	if err != nil {
		return fallback
	}

	return value
}

func serve(data string) error {
	cfg := shareserver.Config{
		Data:       data,
		Difficulty: number("SHARE_DIFFICULTY", 20),
		InFlight:   number("SHARE_IN_FLIGHT", 32),
		Limits: shareserver.Limits{
			PerKeyDay:   number("SHARE_PER_KEY_DAY", 200),
			PerDay:      number("SHARE_PER_DAY", 20_000),
			PerDayBytes: int64(number("SHARE_PER_DAY_MB", 200)) << 20,
			Total:       int64(number("SHARE_TOTAL_MB", 5120)) << 20,
		},
	}

	// SHARE_TRUSTED_PROXIES: addresses or networks of the proxy in front,
	// comma separated (compose sets Caddy's fixed address).
	for _, entry := range strings.Split(env("SHARE_TRUSTED_PROXIES", ""), ",") {
		if entry = strings.TrimSpace(entry); entry == "" {
			continue
		}

		if !strings.Contains(entry, "/") {
			if strings.Contains(entry, ":") {
				entry += "/128"
			} else {
				entry += "/32"
			}
		}

		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return fmt.Errorf("SHARE_TRUSTED_PROXIES: %w", err)
		}

		cfg.Proxies = append(cfg.Proxies, network)
	}

	if secret := env("SHARE_SECRET", ""); secret != "" {
		key, err := hex.DecodeString(secret)
		if err != nil || len(key) < 16 {
			return errors.New("SHARE_SECRET: co najmniej 32 znaki szesnastkowe")
		}

		cfg.Secret = key
	}

	if models := env("SHARE_MODELS", ""); models != "" {
		cfg.Models = map[string]bool{}
		for _, model := range strings.Split(models, ",") {
			cfg.Models[strings.TrimSpace(model)] = true
		}
	}

	server, err := shareserver.New(cfg)
	if err != nil {
		return err
	}
	defer server.Close()

	httpServer := &http.Server{
		Addr:              env("SHARE_ADDR", ":8080"),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go server.Janitor(ctx.Done())

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				server.Sweep()
			}
		}
	}()

	errs := make(chan error, 1)
	go func() { errs <- httpServer.ListenAndServe() }()

	log.Printf("neuro-share na %s, dane w %s, trudność %d", httpServer.Addr, data, cfg.Difficulty)

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return httpServer.Shutdown(shutdown)
}

func verify(data string) int {
	report, err := shareserver.Verify(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 1
	}

	for _, problem := range report.Problems {
		fmt.Println(problem)
	}

	// The heads, kept apart from the data (with a backup, say), show a
	// later cut at the end of a file or a file gone.
	for _, head := range report.Heads {
		fmt.Println("głowa:", head)
	}

	fmt.Printf("plików %d, wpisów %d, problemów %d\n", report.Files, report.Records, len(report.Problems))

	if len(report.Problems) > 0 {
		return 1
	}

	return 0
}
