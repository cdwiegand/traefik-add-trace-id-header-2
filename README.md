# Add Trace Id Header 2

This plugin sets a custom header for tracing to a freshly generated random value on every request, unless the request comes from a trusted IP address or subnet AND already carries that header, in which case the incoming value is kept. Values from untrusted sources are always replaced, so clients cannot inject their own trace IDs.

You can optionally customize the header name (defaults to `X-Trace-Id`), add a prefix and/or suffix around the generated value (both default to empty), and choose the ID format (UUIDv4, UUIDv7 or ULID).

## Credit

Thanks to [trinnylondon/traefik-add-trace-id](https://github.com/trinnylondon/traefik-add-trace-id) for the initial plugin, which I've forked to add trusted IP support, and then modified a little to fit my use case better, and add UUIDv7 and ULID support.

## Configuration

1. Enable the plugin in your Traefik configuration:

```yaml
experimental:
 plugins:
  traceinjector:
   moduleName: github.com/cdwiegand/traefik-add-trace-id-header-2
   version: v0.32.0
```

1. Define the middleware. Note that this plugin does not need any configuration, however, values must be passed in for it to be accepted within Traefik:

```yaml
http:
 # ...
 middlewares:
  # this name must match the middleware that you attach to routers later
  mw-trace-id:
   plugin:
    traceinjector:
     # valuePrefix is prepended to the generated ID
     valuePrefix: ""
     # valueSuffix is appended to the generated ID
     valueSuffix: ""
     # headerName is the HTTP header name to use
     headerName: "X-Trace-Id"
     # uuidGen indicates the type of UUID to generate, 4 being default, 7 being a k-sortable type, L being a ULID
     uuidGen: "4"
     # addToResponse indicates whether to add the header to the response
     addToResponse: true
     # verbose logs each trace ID
     verbose: false
     # trustedIPs lists IP addresses and/or CIDR subnets whose incoming trace header is kept
     # (IPv4 and IPv6 both work; a comma-separated string is also accepted)
     trustedIPs:
      - "10.0.0.0/8"
      - "192.168.1.10"
```

Trust is checked against the address of the peer that connected directly to Traefik, not `X-Forwarded-For`, so a client can't claim to be trusted by setting a header. If Traefik sits behind a load balancer, list the load balancer's addresses. With no `trustedIPs` set, every request gets a new trace ID.

Please note that traefik requires at least one configuration variable set, to keep the defaults you can set `verbose: false` to accommodate this. *This is not a requirement of this plugin, but a traefik requirement.*

1. Then add it to your given routers, such as this:

```yaml
http:
 # ...
 routers:
  example-router:
   rule: host(`demo.localhost`)
   service: service-foo
   entryPoints:
    - web
   # add these 2 lines, use the same name you defined directly under "middlewares":
   middlewares: 
    - mw-trace-id
   # end add those 2 lines
```

1. You are done!

## Testing Methods

Testing by using local plugin functionality, assuming the code is checked out to `C:\devel\traefik-add-trace-id-header-2`:

```bash
docker run --rm -it -p 8888:80 -v C:\devel\traefik-add-trace-id-header-2\:/srv/plugins-local/src/github.com/cdwiegand/traefik-add-trace-id-header-2:ro -w /srv traefik:3.0 --entryPoints.web.address=:80 --experimental.localPlugins.traceIdHeader.modulename=github.com/cdwiegand/traefik-add-trace-id-header-2 --providers.file.filename=/srv/plugins-local/src/github.com/cdwiegand/traefik-add-trace-id-header-2/testing.traefik.yml --api=true --api.dashboard=true
```

and go to <http://localhost:8888/dashboard/> and inspect the browser's Network tab to see the X-Trace-Id header injected in the response.
