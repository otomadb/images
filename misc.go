package main

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseSize(size string) (width, height int, err error) {
	s := strings.Split(size, "x")
	if len(s) != 2 {
		return 0, 0, fmt.Errorf("invalid size")
	}
	if width, err = strconv.Atoi(s[0]); err != nil {
		return 0, 0, fmt.Errorf("invalid width")
	}
	if height, err = strconv.Atoi(s[1]); err != nil {
		return 0, 0, fmt.Errorf("invalid height")
	}

	return width, height, nil
}

func ParseScale(scale string) (width, height int, err error) {
	switch scale {
	case "ogp":
		return 700, 400, nil
	case "large":
		return 960, 720, nil
	default:
		return 0, 0, fmt.Errorf("invalid scale")
	}
}
