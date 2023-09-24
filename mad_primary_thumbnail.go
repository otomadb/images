package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func MADPrimaryThumbnail(c echo.Context) error {
	serial, err := strconv.Atoi(c.Param("serial"))
	if err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusBadRequest, "Bad request")
	}

	scale := c.FormValue("scale")
	width, height, err := ParseScale(scale)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDRESS")})
	defer rdb.Close()

	// Redisにキャッシュがあればリダイレクト
	redisKey := fmt.Sprintf("mad_primary_thumbnail_%d_%d_%d", serial, width, height)
	cachedUrl, err := rdb.Get(ctx, redisKey).Result()
	if err != nil && err != redis.Nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	if cachedUrl != "" {
		return c.Redirect(http.StatusTemporaryRedirect, cachedUrl)
	}

	conn, err := pgx.Connect(ctx, os.Getenv("POSTGRES_URL"))
	if err != nil {
		c.Logger().Error(err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	defer conn.Close(ctx)

	var url string
	q := `SELECT t."imageUrl" FROM "VideoThumbnail" t JOIN "Video" v ON t."videoId" = v."id" WHERE v."serial" = $1 AND t."isPrimary" = TRUE`
	err = conn.QueryRow(ctx, q, serial).Scan(&url)
	if err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	// 画像の存在チェック
	c.Logger().Debug(url)
	resp, err := http.Head(url)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	if resp.StatusCode != http.StatusOK {
		c.Logger().Warn(url, resp.StatusCode)
		return c.String(http.StatusNotFound, "Not Found")
	}

	proxiedUrl := SignURL(
		fmt.Sprintf("/rs:fit:%d:%d:1:1/background:000000/%s", width, height, base64.URLEncoding.EncodeToString([]byte(url))))
	c.Logger().Debug(proxiedUrl)

	// Redisにキャッシュ
	err = rdb.Set(ctx, redisKey, proxiedUrl, time.Duration(600*time.Second)).Err()
	if err != nil {
		c.Logger().Error(err)
	}

	return c.Redirect(http.StatusTemporaryRedirect, proxiedUrl)
}
