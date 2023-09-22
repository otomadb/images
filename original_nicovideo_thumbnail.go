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

func OriginalNicovideoThumbnail(c echo.Context) error {
	vid := c.Param("vid")
	if vid == "" {
		c.String(http.StatusBadRequest, "Bad request")
	}

	size := c.FormValue("size")
	if size == "" {
		c.String(http.StatusBadRequest, "Bad request")
	}
	width, height, err := ParseSize(size)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
	}

	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDRESS")})
	defer rdb.Close()

	redisKey := fmt.Sprintf("original_nicovideo_thumbnail_url_%s_%d_%d", vid, width, height)
	cachedUrl, err := rdb.Get(ctx, redisKey).Result()

	if err != nil && err != redis.Nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	if cachedUrl != "" {
		return c.Redirect(http.StatusTemporaryRedirect, cachedUrl)
	}

	apiUrl := fmt.Sprintf("https://www.nicovideo.jp/api/watch/v3_guest/%s?&_frontendId=6&_frontendVersion=0&skips=harmful&actionTrackId=0_0", vid)
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
			Video struct {
				Thumbnail struct {
					Ogp string `json:"ogp"`
				} `json:"thumbnail"`
			} `json:"video"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		c.Logger().Fatal(err)
		return c.String(http.StatusNotFound, "Not Found")
	}

	c.Logger().Debug(data.Data.Video.Thumbnail.Ogp)

	proxiedUrl := SignURL(
		fmt.Sprintf("/rs:fit:%d:%d:1:1/background:000000/%s", width, height, base64.URLEncoding.EncodeToString([]byte(data.Data.Video.Thumbnail.Ogp))))
	c.Logger().Debug(proxiedUrl)

	err = rdb.Set(ctx, redisKey, proxiedUrl, time.Duration(600*time.Second)).Err()
	if err != nil {
		c.Logger().Error(err)
	}

	return c.Redirect(http.StatusTemporaryRedirect, proxiedUrl)
}
