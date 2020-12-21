package rendertron

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

var (
	defaultUserAgentPatern   = regexp.MustCompile(strings.Join(botUserAgents[:], "|"))
	defaultExcludeUrlPattern = regexp.MustCompile("\\.(" + strings.Join(staticFileExtensions[:], "|") + ")$")
)

func init() {
	caddy.RegisterModule(Rendertron{})
}

var (
	_ caddy.Provisioner           = (*Rendertron)(nil)
	_ caddyhttp.MiddlewareHandler = (*Rendertron)(nil)
	_ caddyfile.Unmarshaler       = (*Rendertron)(nil)
	_ caddy.Validator             = (*Rendertron)(nil)
)

type Rendertron struct {
	ExcludeUrlPattern string `json:"excludeUrlPattern,omitempty"`
	UserAgentPattern  string `json:"userAgentPattern,omitempty"`

	Timeout caddy.Duration `json:"timeout,omitempty"`
	Proxy   string         `json:"proxy"`

	AllowedForwadedHosts []string `json:"allowedForwadedHosts,omitempty"`
	ForwardedHostHeader  string   `json:"forwardedHostHeader,omitempty"`

	log               *zap.Logger
	cli               RendertronClient
	excludeUrlPattern *regexp.Regexp
	userAgentPattern  *regexp.Regexp
}

func (Rendertron) CaddyModule() caddy.ModuleInfo {

	return caddy.ModuleInfo{
		ID:  "http.handlers.rendertron",
		New: func() caddy.Module { return new(Rendertron) },
	}
}

func (c *Rendertron) Provision(ctx caddy.Context) error {
	c.log = ctx.Logger(c)

	pu, err := url.Parse(c.Proxy)
	if err != nil {
		return fmt.Errorf("specified proxy is not a valid url: %w", err)
	}

	if pu.Scheme != "chrome" {
		c.cli = NewProxyRendertronClient(c.Proxy)
	}

	if c.ExcludeUrlPattern == "" {
		c.excludeUrlPattern = defaultExcludeUrlPattern
	} else {
		re, err := regexp.Compile(c.ExcludeUrlPattern)
		if err != nil {
			return fmt.Errorf("unable to parse exclude pattern: %w", err)
		}
		c.excludeUrlPattern = re
	}

	if c.UserAgentPattern == "" {
		c.userAgentPattern = defaultUserAgentPatern
	} else {
		re, err := regexp.Compile(c.UserAgentPattern)
		if err != nil {
			return fmt.Errorf("unable to parse user agent pattern: %w", err)
		}
		c.userAgentPattern = re
	}

	if c.Timeout == 0 {
		// The Rendertron service itself has a hard limit of 10 seconds to render, so
		// let's give a little more time than that by default.
		c.Timeout = 11 * caddy.Duration(time.Second)
	}

	if c.ForwardedHostHeader == "" {
		c.ForwardedHostHeader = "X-Forwarded-Host"
	}

	return nil
}

func (c *Rendertron) Validate() error {
	if c.excludeUrlPattern == nil {
		return errors.New("exclude pattern is nil")
	}

	if c.userAgentPattern == nil {
		return errors.New("user agent pattern is nil")
	}

	if c.cli == nil {
		return errors.New("render mechanishm is not selected please select it using the proxy option")
	}

	return nil
}

func (c *Rendertron) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	ua := r.Header.Get("User-Agent")
	if ua == "" || !c.userAgentPattern.MatchString(ua) || c.excludeUrlPattern.MatchString(ua) {
		return next.ServeHTTP(w, r)
	}

	host := r.Host
	hh := r.Header.Get(c.ForwardedHostHeader)
	for _, h := range c.AllowedForwadedHosts {
		if h == hh {
			host = hh
			break
		}
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout))
	defer cancel()

	ru := scheme + "://" + host + r.RequestURI
	res, err := c.cli.Render(ctx, ru, nil)
	if err != nil {
		c.log.Error("unable to render page", zap.String("url", ru), zap.Error(err))
		return err
	}
	defer res.Body.Close()

	// Copy headers
	for k, vv := range res.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(res.StatusCode)
	if _, err := io.Copy(w, res.Body); err != nil {
		c.log.Error("error writing response to client", zap.Error(err))
		return err
	}

	return nil
}

var (
	botUserAgents = [...]string{
		"Baiduspider",
		"bingbot",
		"Embedly",
		"facebookexternalhit",
		"LinkedInBot",
		"outbrain",
		"pinterest",
		"quora link preview",
		"rogerbot",
		"showyoubot",
		"Slackbot",
		"TelegramBot",
		"Twitterbot",
		"vkShare",
		"W3C_Validator",
		"WhatsApp",
	}

	staticFileExtensions = [...]string{
		"ai",
		"avi",
		"css",
		"dat",
		"dmg",
		"doc",
		"doc",
		"exe",
		"flv",
		"gif",
		"ico",
		"iso",
		"jpeg",
		"jpg",
		"js",
		"less",
		"m4a",
		"m4v",
		"mov",
		"mp3",
		"mp4",
		"mpeg",
		"mpg",
		"pdf",
		"png",
		"ppt",
		"psd",
		"rar",
		"rss",
		"svg",
		"swf",
		"tif",
		"torrent",
		"ttf",
		"txt",
		"wav",
		"wmv",
		"woff",
		"xls",
		"xml",
		"zip",
	}
)
