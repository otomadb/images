package main

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var ctx = context.Background()

func main() {
	e := echo.New()
	e.Debug = true
	e.Use(middleware.Logger())

	e.GET("/original/youtube/:vid", OriginalYoutubeThumbnail)
	e.GET("/original/nicovideo/:vid", OriginalNicovideoThumbnail)
	e.GET("/original/bilibili/:vid", OriginalBilibiliThumbnail)

	e.Logger.Fatal(e.Start(":1323"))
}
