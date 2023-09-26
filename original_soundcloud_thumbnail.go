package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	sc "github.com/zackradisic/soundcloud-api"
)

func OriginalSoundcloudThumbnail(c echo.Context) error {
	queryId := c.FormValue("id")
	queryUrl := c.FormValue("url")

	if queryId == "" && queryUrl == "" {
		return c.String(http.StatusBadRequest, "Bad request")
	}
	if queryId != "" && queryUrl != "" {
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

	imgRedisKey := fmt.Sprintf("original_soundcloud_thumbnail_url_%s_%d_%d", queryUrl, width, height)
	cachedImgUrl, err := rdb.Get(ctx, imgRedisKey).Result()
	if err != nil && err != redis.Nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	if cachedImgUrl != "" {
		c.Logger().Debug(cachedImgUrl)
		return c.Redirect(http.StatusTemporaryRedirect, cachedImgUrl)
	}

	// SoundCloudのClientIDを取得
	clientIdRedisKey := "soundcloud_client_id"
	clientId, err := rdb.Get(ctx, clientIdRedisKey).Result()
	if err != nil && err != redis.Nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	if clientId == "" {
		clientId, err = sc.FetchClientID()
		if err != nil {
			c.Logger().Error(err)
			c.String(http.StatusInternalServerError, "Internal Server Error")
		}
	}
	c.Logger().Debug(clientId)

	apiUrl := fmt.Sprintf("https://api-v2.soundcloud.com/resolve?url=%s&client_id=%s", queryUrl, clientId)
	resp, err := http.Get(apiUrl)
	if err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	if resp.StatusCode != http.StatusOK {
		c.Logger().Error(apiUrl)
		return c.String(http.StatusBadRequest, "Request Failed")
	}

	defer resp.Body.Close()
	var data struct {
		ArtworkUrl string `json:"artwork_url"`
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		c.Logger().Fatal(err)
		return c.String(http.StatusNotFound, "Not Found")
	}

	imgUrl := strings.Replace(data.ArtworkUrl, "-large", "-t500x500", -1)
	c.Logger().Debug(imgUrl)

	proxiedImgUrl := SignURL(
		fmt.Sprintf("/rs:fit:%d:%d:1:1/background:000000/%s", width, height, base64.URLEncoding.EncodeToString([]byte(imgUrl))))
	c.Logger().Debug(proxiedImgUrl)

	// プロキシした画像Urlをキャッシュ
	err = rdb.Set(ctx, imgRedisKey, proxiedImgUrl, time.Duration(600*time.Second)).Err()
	if err != nil {
		c.Logger().Error(err)
	}

	// SoundCloudのClientIDをキャッシュ
	err = rdb.Set(ctx, clientIdRedisKey, clientId, time.Duration(12*time.Hour)).Err()
	if err != nil {
		c.Logger().Error(err)
	}

	return c.Redirect(http.StatusTemporaryRedirect, proxiedImgUrl)
}
