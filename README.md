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

## Limitations
**Inbound does not validate incoming mail!** This is to say, no SPF, DKIM or
ARC checks are done. Inbound can receive mail from a trusted server
(over internal network, Internet with IP restrictions, etc.) and make it
available to your application. Running it as Internet-facing mail server is
*not* recommended unless you really don't care about sender identity.

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

To run Inbound, point it to S3-like or local storage:
```sh
inbound -local-dir /path/to/maildir
inbound -s3-bucket my-unique-bucket-name
```

If you're not on AWS, credentials for S3 can be specified using
environment variables:
```sh
AWS_PROFILE=my-profile inbound -s3-bucket my-unique-bucket-name
AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... -s3-bucket my-unique-bucket-name
```
The latter approach also works if your *bucket* is not on AWS. For example of
this usage, see `test.sh`.

If you intend to receive *large* files, note that `-max-size` applies for
MIME-encoded for of mail! In other words, no, you cannot receive
100mb (or even 90mb) attachments without increasing `-max-size`.

### STARTTLS
In year 2025, what runs unencrypted over Internet? Email (and DNS), of course!
Thankfully, only by default; most mail servers support STARTTLS for encryption.
Inbound is no exception:

```sh
./inbound -s3-bucket my-unique-bucket-name -tls-cert server.crt -tls-key server.key
```

Or, if you're running on a platform where injecting secrets to environment
variables is easier than putting them to files:
```sh
# Be sure to set INBOUND_TLS_CERT and INBOUND_TLS_KEY
./inbound -s3-bucket my-unique-bucket-name -tls-from-env
```

The keys do not necessarily need to be trusted by any CA (unless you use MTA-STS).
Creating self-signed ones looks something like this (see `test.sh`):
```sh
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 \
    -keyout test/server.key -out test/server.crt -nodes \
    -days 365 -sha256 \
    -subj "/CN=localhost" \
    -addext "subjectAltName=DNS:localhost"
```

You should probably use either DANE or MTA-STS. Both should work with Inbound;
it is the sender who verifies these things.

## Ingesting mail
So you've installed Inbound and sent some mail. How to read it?

The rough outline is:
1. List content of `/messages` in your `-s3-bucket`
   * If you set `-s3-prefix`, add it to start of prefix
2. Fetch and read each JSON file one-by one. Load them to your database
3. If you ever need attachments, they can be found under `/attachments/`

A few caveats:
* **From and To fields are not to be trusted**
  * From can be trusted only if Inbound is behind another mail server that
    *rejects* mail with invalid SPF, DKIM, ARC, etc.
  * To can be anything, Inbound doesn't care
* **Beware of XSS** - none of the fields are sanitized for bad HTML
  * NEVER dangerously set `innerHTML` without using a good sanitazion
    [library](https://github.com/cure53/DOMPurify)
  * Content-Type is a lie. Well, not always, but it *can* be. `text/plain`
    might or might not actually be HTML
  * Beware of path traversal attacks; never use `original_filename`s
    of attachments as their *actual* file names!

### Message schema
```jsonc
{
    "id": "string", // Unique id for message, generated by Inbound
    "from": "string", // From field - BEWARE, not necessarily validated!
    "to": "string", // To field - Inbound doesn't do anything with this, or validate it
    "subject": "string", // Subject line
    "content": "string", // Main message content - beware of XSS!
    "content_type": "string", // Content-Type header's value; client may have lied!
    "attachments": [
        // Attachments can be found in $prefix/attachments/$id
        {
            "id": "string", // Unique id for attachment
            "original_filename": "string" // For display purposes only, client may have lied!
        }
    ]
}
```