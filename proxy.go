package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func Proxy(c echo.Context) error {
	url := c.FormValue("url")
	if url == "" {
		return c.String(http.StatusBadRequest, "Bad request at url")
	}

	scale := c.FormValue("scale")
	if scale == "" {
		return c.String(http.StatusBadRequest, "Bad request at scale")
	}
	width, height, err := ParseScale(scale)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDRESS")})
	defer rdb.Close()

	redisKey := fmt.Sprintf("proxy_%s_%d_%d", url, width, height)
	cachedUrl, err := rdb.Get(ctx, redisKey).Result()
	if err != nil && err != redis.Nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	if cachedUrl != "" {
		return c.Redirect(http.StatusTemporaryRedirect, cachedUrl)
	}

	proxiedUrl := SignURL(
		fmt.Sprintf("/rs:fit:%d:%d:1:1/background:000000/%s", width, height, base64.URLEncoding.EncodeToString([]byte(url))))
	c.Logger().Debug(proxiedUrl)

	err = rdb.Set(ctx, redisKey, proxiedUrl, time.Duration(600*time.Second)).Err()
	if err != nil {
		c.Logger().Error(err)
	}

	return c.Redirect(http.StatusTemporaryRedirect, proxiedUrl)
}
