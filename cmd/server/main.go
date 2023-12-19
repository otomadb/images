package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	proxyv2 "otomadb.com/ixgyohn/gen/proxy/v2"
	"otomadb.com/ixgyohn/gen/proxy/v2/proxyv2connect"
	"otomadb.com/ixgyohn/pkg/imgproxy"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

type ProxyServer struct{}

type ProxyThisOption struct {
	Width  uint32
	Height uint32
}

func proxythis(ctx context.Context, inputUrl string, option ProxyThisOption) (string, error) {
	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDRESS")})
	defer rdb.Close()

	redisKey := fmt.Sprintf("ixgyohn:proxiedurl_%s_%d_%d", inputUrl, option.Width, option.Height)
	cachedUrl, err := rdb.Get(ctx, redisKey).Result()
	if err != nil && err != redis.Nil {
		logger.Warn("Failed to get from redis", "error", err)
	}
	if cachedUrl != "" {
		return cachedUrl, nil
	}

	// check url is valid
	u, err := url.ParseRequestURI(inputUrl)
	if err != nil {
		logger.Error("URL is not valid", "input", inputUrl, "error", err)
		return "", err
	}

	// TODO: 本来なら失敗したならここで打ち切るべきだが結構な確率で失敗するのでとりあえず警告を出すに留める．
	resp, err := http.Head(u.String())
	if err != nil {
		logger.Warn("URL is not accessible", "url", u.String(), "error", err)
	} else if resp.StatusCode != http.StatusOK {
		logger.Warn("URL is not accessible", "url", u.String())
	} else if resp.Header.Get("Content-Type") != "image/jpeg" && resp.Header.Get("Content-Type") != "image/png" {
		logger.Warn("URL may not be image", "url", u.String(), "content-type", resp.Header.Get("Content-Type"))
	}

	path := fmt.Sprintf("/rs:fit:%d:%d:1:1/background:000000/%s", option.Width, option.Height, base64.URLEncoding.EncodeToString([]byte(u.String())))
	proxiedUrl, err := imgproxy.SignPath(path)
	if err != nil {
		logger.Error("Failed to sign path", "url", u.String(), "path", path, "error", err)
		return "", err
	}

	err = rdb.Set(ctx, redisKey, proxiedUrl, time.Duration(24*time.Hour)).Err()
	if err != nil {
		logger.Warn("Failed to set redis", "error", err)
	}

	return proxiedUrl, nil
}

func (s *ProxyServer) ProxyUrl(
	ctx context.Context,
	req *connect.Request[proxyv2.ProxyUrlRequest],
) (*connect.Response[proxyv2.ProxyUrlResponse], error) {
	logger.Info("Requested", "header", req.Header())

	proxiedUrl, err := proxythis(ctx, req.Msg.Url, ProxyThisOption{
		Width:  req.Msg.Width,
		Height: req.Msg.Height,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	res := connect.NewResponse(&proxyv2.ProxyUrlResponse{ProxiedUrl: proxiedUrl})
	res.Header().Set("Proxy-Version", "v2")
	return res, nil
}

func main() {
	proxy := &ProxyServer{}
	mux := http.NewServeMux()
	path, handler := proxyv2connect.NewProxyServiceHandler(proxy)
	mux.Handle(path, handler)

	logger.Info("Server started", "port", 38080)
	if err := http.ListenAndServe(
		"0.0.0.0:38080",
		h2c.NewHandler(mux, &http2.Server{}),
	); err != nil {
		logger.Error("Failed to start server", "error", err)
		panic(err)
	}
}
