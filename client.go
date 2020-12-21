package rendertron

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

var (
	_ RendertronClient = (*ProxyRendertronClient)(nil)
)

type RendertronClient interface {
	Render(ctx context.Context, u string, opt *Options) (*http.Response, error)
}

type Options struct {
	InjectShadyDom bool
}

func (o *Options) GetInjectShadyDom() bool {
	if o == nil {
		return false
	}

	return o.InjectShadyDom
}

type ProxyRendertronClient struct {
	proxy  string
	client *http.Client
}

func NewProxyRendertronClient(proxy string) *ProxyRendertronClient {
	if proxy[len(proxy)-1] != '/' {
		proxy += "/"
	}

	proxy += "render/"

	return &ProxyRendertronClient{
		proxy:  proxy,
		client: &http.Client{},
	}
}

func (cli *ProxyRendertronClient) Render(ctx context.Context, u string, opt *Options) (*http.Response, error) {
	ru := cli.proxy + url.PathEscape(u)
	if opt.GetInjectShadyDom() {
		ru += "?wc-inject-shadydom=true"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ru, nil)
	if err != nil {
		return nil, fmt.Errorf("redertron: unable to create request: %w", err)
	}

	return cli.client.Do(req)
}
