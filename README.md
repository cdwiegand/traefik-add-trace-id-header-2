# About

This plugin will append a custom header for tracing with a random value unless the remote IP is trusted AND there is already a specified trace header present in the incoming request.

You can optionally customise this by specifying a custom header name that the plugin will look for in the incoming request (defaults to `X-Trace-Id`) and you can also specify a custom prefix to be added to that value (defaults to no prefix).

# Credit

Thanks to [trinnylondon/traefik-add-trace-id](https://github.com/trinnylondon/traefik-add-trace-id) for the initial plugin, which I've forked to add trusted IP support, and then modified a little to fit my use case better, and add UUID gen 7 support.

# Configuration
1. Enable the plugin in your Traefik configuration:

```
experimental:
	plugins:
		traceinjector:
			moduleName: github.com/cdwiegand/traefik-add-trace-id
			version: v0.2.1
```

2. Define the middleware. Note that this plugin does not need any configuration, however, values must be passed in for it to be accepted within Traefik:
```
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
					# trustNetworks specifies networks in CIDR notation to trust, leaving an existing trace header if found
					trustNetworks: "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,fc00::/7,fec0::/10"
					# trustAllIPs means we leave any existing trace header if found
					trustAllIPs: false
					# trustAllPrivateIPs allows private networks to leave an existing trace header if found
					trustAllPrivateIPs: false	 # this also includes localhost
					# trustLocalhost trusts only localhost (127.0.0.0/24 or ::1) to leave an existing trace header if found
					trustLocalhost: false			 # this means only 127.0.0.0/24 and ::1
					# uuidGen indicates the type of UUID to generate, 4 being default, 7 being a k-sortable type
					uuidGen: 4
```
Please note that traefik requires at least one configuration variable set, to keep the defaults you can set `trustAllIPs: false` to accomodate this. *This is not a requirement of this plugin, but a traefik requirement.*

3. Then add it to your given routers, such as this:
```
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

4. You are done!