package main

import (
	"flag"
	"log/slog"

	"github.com/solita/inbound/core"
	"github.com/solita/inbound/sinks"
)

func main() {
	localDir := flag.String("local-dir", "", "Local directory to store mail to")
	s3Bucket := flag.String("s3-bucket", "", "S3 bucket to store mail to")
	s3Prefix := flag.String("s3-prefix", "", "S3 prefix inside the bucket")

	listenAddr := flag.String("listen", "localhost:1025", "Address to listen for incoming mail")
	domain := flag.String("domain", "localhost", "Domain to identify this server in SMTP greetings")
	maxMessageMb := flag.Int("max-message", 100, "Maximum size of an incoming message in megabytes")

	flag.Parse()

	enabledSinks := []core.Sink{&sinks.LoggingSink{}}
	if *localDir != "" {
		localSink, err := sinks.NewLocal(*localDir)
		if err != nil {
			slog.Error("Failed to create local sink", "error", err)
			return
		}
		enabledSinks = append(enabledSinks, localSink)
	}
	if *s3Bucket != "" {
		s3Sink, err := sinks.NewS3(*s3Bucket, *s3Prefix)
		if err != nil {
			slog.Error("Failed to create S3 sink", "error", err)
			return
		}
		enabledSinks = append(enabledSinks, s3Sink)
	}

	server := core.NewServer(enabledSinks)
	server.Addr = *listenAddr
	server.Domain = *domain
	server.MaxMessageBytes = int64(*maxMessageMb) * 1024 * 1024

	slog.Info("Starting inbound incoming mail server")
	err := server.ListenAndServe()
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
}
