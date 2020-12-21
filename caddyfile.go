package rendertron

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("redertron", parseCaddyfile)
}

// parseCaddyfile sets up the handler from Caddyfile tokens. Syntax:
//
//		redertron [<matcher>] <url> {
//			excludeUrlPattern <pattern>
// 			userAgentPattern <pattern>
//			timeout <timeout>
//			proxy <url>
// 			allowedForwadedHosts <text...>
// 			forwardedHostHeader <text=X-Forwarded-Host>
//		}
//
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	t := new(Rendertron)
	if err := t.UnmarshalCaddyfile(h.Dispenser); err != nil {
		return nil, err
	}
	return t, nil
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler. Syntax:
//
//		redertron [<matcher>] <url> {
//			excludeUrlPattern <pattern>
// 			userAgentPattern <pattern>
//			timeout <timeout>
//			proxy <url>
// 			allowedForwadedHosts <text...>
// 			forwardedHostHeader <text=X-Forwarded-Host>
//		}
//
func (c *Rendertron) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.Args(&c.Proxy) {
			return d.ArgErr()
		}

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "excludeUrlPattern":
				if !d.Args(&c.ExcludeUrlPattern) {
					return d.ArgErr()
				}
			case "userAgentPattern":
				if !d.Args(&c.UserAgentPattern) {
					return d.ArgErr()
				}
			case "timeout":
				var ts string
				if !d.Args(&ts) {
					return d.ArgErr()
				}

				timeout, err := caddy.ParseDuration(ts)
				if err != nil {
					return d.Errf("unable to parse timeout: %v", err)
				}
				c.Timeout = caddy.Duration(timeout)

			case "proxy":
				if c.Proxy != "" {
					return d.Err("proxy specified twice")
				}

				if !d.Args(&c.Proxy) {
					return d.ArgErr()
				}
			case "allowedForwadedHosts":
				c.AllowedForwadedHosts = d.RemainingArgs()
			case "forwardedHostHeader":
				if !d.Args(&c.ForwardedHostHeader) {
					return d.ArgErr()
				}
			}
		}

		if d.Next() {
			return d.Errf("unknown argument: %v", d.Val())
		}
	}
	return nil
}
