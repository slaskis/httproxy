# httproxy

A single binary to proxy traffic based on paths.

## Example

If your app, running on port 8000, has a separate server for health checks running on port 4040.

```
httproxy /health=:4040 /=:8000
```

## Docker

`httproxy` can be used as a Docker `ENTRYPOINT` by ending the arguments with `--`. Then it will run all the arguments after as a command in a subprocess which when shutdown will automatically turn off the proxy.

Here's an example using a separate build step for the download and unpacking to keep the final image lean.

```dockerfile
FROM alpine:3.11.6 AS download
RUN apk --update add ca-certificates
RUN mkdir -m 777 /scratchtmp
RUN wget https://github.com/slaskis/httproxy/releases/download/0.4.0/httproxy-0.4.0-linux-amd64.tar.gz
RUN tar -xzf httproxy-0.4.0-linux-amd64.tar.gz

FROM alpine:3.11.6
COPY --from=download /scratchtmp /tmp
COPY --from=download /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=download /httproxy /usr/local/bin/httproxy
ENTRYPOINT ["httproxy", "/health/=localhost:13133/", "/=localhost:55681", "--"]
CMD ["app"]
```
