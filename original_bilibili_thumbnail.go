package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func OriginalBilibiliThumbnail(c echo.Context) error {
	vid := c.Param("vid")
	if vid == "" {
		return c.String(http.StatusBadRequest, "Bad request")
	}

	scale := c.FormValue("scale")
	if scale == "" {
		return c.String(http.StatusBadRequest, "Bad request")
	}
	width, height, err := ParseScale(scale)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDRESS")})
	defer rdb.Close()

	redisKey := fmt.Sprintf("original_bilibili_thumbnail_url_%s_%d_%d", vid, width, height)
	cachedUrl, err := rdb.Get(ctx, redisKey).Result()
	if err != nil && err != redis.Nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	if cachedUrl != "" {
		return c.Redirect(http.StatusTemporaryRedirect, cachedUrl)
	}

	apiUrl := fmt.Sprintf("https://api.bilibili.com/x/web-interface/view/detail?bvid=%s", vid)
	resp, err := http.Get(apiUrl)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	if resp.StatusCode != http.StatusOK {
		c.Logger().Fatal(apiUrl)
		return c.String(http.StatusNotFound, "Not Found")
	}

	defer resp.Body.Close()
	var data struct {
		Data struct {
			View struct {
				Thumbnail string `json:"pic"`
			} `json:"View"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		c.Logger().Fatal(err)
		return c.String(http.StatusNotFound, "Not Found")
	}

	c.Logger().Debug(vid, data.Data.View.Thumbnail)

	proxiedUrl := SignURL(
		fmt.Sprintf("/rs:fit:%d:%d:1:1/background:000000/%s", width, height, base64.URLEncoding.EncodeToString([]byte(data.Data.View.Thumbnail))))
	c.Logger().Debug(proxiedUrl)

	err = rdb.Set(ctx, redisKey, proxiedUrl, time.Duration(600*time.Second)).Err()
	if err != nil {
		c.Logger().Error(err)
	}

	return c.Redirect(http.StatusTemporaryRedirect, proxiedUrl)
}
