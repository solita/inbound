# inbound
Inbound is an email server for piping incoming mail to S3-compatible object
storage (or to a local directory).

Disclaimer: Not a supported Solita project. Solitans can ask @bensku for help.

## Why?
If you *need* to receive email, and
* Don't want to add an SMTP server to your application
* Want email and its attachments as separate files, not in raw MIME format
* Need to receive large(r than what AWS SES permits) emails

... Inbound might be an useful tool for you.

## Building
```sh
go build # produces inbound executable at repo root
```

Or a container image:
```sh
podman build -t inbound .
```

Or if you want to crosscompile, *then* build a container image for another
arch (which is much faster than multistage build):
```sh
GOARCH=arm64 go build
podman build -f Dockerfile.plain --arch arm64 -t inbound:arm .
```

To run tests, use `test.sh`. You'll need `swaks` email testing tool installed.

## Usage
```
Usage of ./inbound:
  -domain string
        Domain to identify this server in SMTP greetings (default "localhost")
  -listen string
        Address to listen for incoming mail (default "localhost:1025")
  -local-dir string
        Local directory to store mail to
  -max-size int
        Maximum size of an incoming message in megabytes (default 100)
  -s3-bucket string
        S3 bucket to store mail to
  -s3-endpoint string
        S3 base endpoint URL (for non-AWS object storage)
  -s3-prefix string
        S3 prefix inside the bucket
  -tls-cert string
        Path to TLS certificate file
  -tls-from-env
        Load TLS certificate from INBOUND_TLS_CERT and private key from INBOUND_TLS_KEY environment variables
  -tls-key string
        Path to TLS private key file
```