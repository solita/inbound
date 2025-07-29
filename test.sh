#!/bin/bash
set -euo pipefail

# Somewhat cursed test script that makes sure inbound puts files in correct places

# Configuration that matches s3mock
export AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"

go build
trap 'kill $(jobs -p)' EXIT # Stop background jobs on exit
# Run inbound server on background
./inbound -s3-endpoint http://localhost:9090 -s3-bucket inbound-files -max-size 150 &

# Create some test attachments
head -c 1M </dev/urandom >test/small.bin
head -c 10M </dev/urandom >test/medium.bin
head -c 100M </dev/urandom >test/large.bin

swaks_cmd="swaks --silent --to test@localhost --from foo@example.com --server localhost:1025"
# s3_get="aws s3api get-object --bucket inbound-files --endpoint-url=http://localhost:9090 --key"

set -x

# Simple mail without attachments
$swaks_cmd --header "Subject: test mail" --body "no attachments"
grep -qr '"content":"no attachments"' test/s3root
grep -qr '"subject":"test mail"' test/s3root
grep -qr '"content_type":"text/plain"' test/s3root

# HTML mail
$swaks_cmd --header "Subject: HTML mail" --header "Content-Type: text/html" --body "<p>no attachments</p>"
grep -qr '"content":"\\u003cp\\u003eno attachments\\u003c/p\\u003e"' test/s3root
grep -qr '"subject":"HTML mail"' test/s3root
grep -qr '"content_type":"text/html"' test/s3root

# Attachments
$swaks_cmd --header "Subject: Small attachment" --body "small attachment" --attach @test/small.bin
grep -qr '"content":"small attachment"' test/s3root
grep -qr '"subject":"Small attachment"' test/s3root
grep -qr '"content_type":"text/plain"' test/s3root
grep -qr '"original_filename":"small.bin"' test/s3root
# Shell magic: check that file content matches byte for byte
find test/s3root -type f -exec sh -c 'cmp -s "$0" "$1" && { printf "%s\n" "$1"; exit 0; }' test/small.bin {} \; -print -quit

$swaks_cmd --header "Subject: Medium attachment" --body "medium attachment" --attach @test/medium.bin
grep -qr '"content":"medium attachment"' test/s3root
grep -qr '"subject":"Medium attachment"' test/s3root
grep -qr '"content_type":"text/plain"' test/s3root
grep -qr '"original_filename":"medium.bin"' test/s3root
find test/s3root -type f -exec sh -c 'cmp -s "$0" "$1" && { printf "%s\n" "$1"; exit 0; }' test/medium.bin {} \; -print -quit

$swaks_cmd --header "Subject: Large attachment" --body "large attachment" --attach @test/large.bin
grep -qr '"content":"large attachment"' test/s3root
grep -qr '"subject":"Large attachment"' test/s3root
grep -qr '"content_type":"text/plain"' test/s3root
grep -qr '"original_filename":"large.bin"' test/s3root
find test/s3root -type f -exec sh -c 'cmp -s "$0" "$1" && { printf "%s\n" "$1"; exit 0; }' test/large.bin {} \; -print -quit