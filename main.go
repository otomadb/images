package main

import (
	"context"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var ctx = context.Background()

func main() {

	e := echo.New()

	e.Debug = os.Getenv("DEBUG") == "true"

	e.Use(middleware.Logger())

	e.GET("/mads/:serial/primary", MADPrimaryThumbnail)

	e.GET("/proxy", Proxy)

	e.GET("/original/youtube/:vid", OriginalYoutubeThumbnail)
	e.GET("/original/nicovideo/:vid", OriginalNicovideoThumbnail)
	e.GET("/original/bilibili/:vid", OriginalBilibiliThumbnail)
	e.GET("/original/soundcloud", OriginalSoundcloudThumbnail)

	e.Logger.Fatal(e.Start(":1323"))
}
