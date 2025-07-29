package main

import (
	"crypto/tls"
	"flag"
	"log/slog"

	"github.com/solita/inbound/core"
	"github.com/solita/inbound/sinks"
)

func main() {
	localDir := flag.String("local-dir", "", "Local directory to store mail to")
	s3Bucket := flag.String("s3-bucket", "", "S3 bucket to store mail to")
	s3Prefix := flag.String("s3-prefix", "", "S3 prefix inside the bucket")
	s3Endpoint := flag.String("s3-endpoint", "", "S3 base endpoint URL (for non-AWS object storage)")

	listenAddr := flag.String("listen", "localhost:1025", "Address to listen for incoming mail")
	domain := flag.String("domain", "localhost", "Domain to identify this server in SMTP greetings")
	maxSizeMb := flag.Int("max-size", 100, "Maximum size of an incoming message in megabytes")

	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file")
	tlsKey := flag.String("tls-key", "", "Path to TLS private key file")

	flag.Parse()

	enabledSinks := []core.Sink{&sinks.LoggingSink{}}
	if *localDir != "" {
		slog.Info("Storing mail to local directory", "directory", *localDir)
		localSink, err := sinks.NewLocal(*localDir)
		if err != nil {
			slog.Error("Failed to create local sink", "error", err)
			return
		}
		enabledSinks = append(enabledSinks, localSink)
	}
	if *s3Bucket != "" {
		slog.Info("Storing mail to S3", "bucket", *s3Bucket, "prefix", *s3Prefix, "endpoint", *s3Endpoint)
		s3Sink, err := sinks.NewS3(*s3Bucket, *s3Prefix, *s3Endpoint)
		if err != nil {
			slog.Error("Failed to create S3 sink", "error", err)
			return
		}
		enabledSinks = append(enabledSinks, s3Sink)
	}

	slog.Info("Setting up mail server")
	server := core.NewServer(enabledSinks)
	server.Addr = *listenAddr
	server.Domain = *domain
	server.MaxMessageBytes = int64(*maxSizeMb) * 1024 * 1024

	if *tlsCert != "" {
		slog.Info("STARTTLS support enabled")
		cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
		if err != nil {
			slog.Error("Failed to load TLS certificate and key", "error", err)
			return
		}
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	} else {
		slog.Warn("Certificate not present, STARTTLS will fail!")
	}

	slog.Info("Starting mail server", "address", *listenAddr)
	err := server.ListenAndServe()
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
}
