# Add Trace Id Header 2

This plugin will append a custom header for tracing with a random value unless the remote IP is trusted AND there is already a specified trace header present in the incoming request.

You can optionally customise this by specifying a custom header name that the plugin will look for in the incoming request (defaults to `X-Trace-Id`) and you can also specify a custom prefix to be added to that value (defaults to no prefix).

## Credit

Thanks to [trinnylondon/traefik-add-trace-id](https://github.com/trinnylondon/traefik-add-trace-id) for the initial plugin, which I've forked to add trusted IP support, and then modified a little to fit my use case better, and add UUID gen 7 support.

## Configuration

1. Enable the plugin in your Traefik configuration:

```yaml
experimental:
 plugins:
  traceinjector:
   moduleName: github.com/cdwiegand/traefik-add-trace-id-2
   version: v0.2.1
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
     # valuePrefix is prepended to the generated GUID
     valuePrefix: ""
     # headerName is the HTTP header name to use
     headerName: "X-Trace-Id"
     # uuidGen indicates the type of UUID to generate, 4 being default, 7 being a k-sortable type, L being a ULID
     uuidGen: 4
```

Please note that traefik requires at least one configuration variable set, to keep the defaults you can set `trustAllIPs: false` to accomodate this. *This is not a requirement of this plugin, but a traefik requirement.*

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
