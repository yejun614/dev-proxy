package main

import (
	"os"
    "log"
    "fmt"
    "strings"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/proxy"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/gofiber/fiber/v2/middleware/favicon"
)

const (
	VERSION = "v0.1.1"
)

var (
	App *fiber.App
	Flags map[string]string
	Addr string = "localhost:8000"
)

func help() {
	fmt.Printf("Dev Proxy (%s)\n", VERSION)
	fmt.Printf(" : A solution that addresses security issues like CORS\n")
	fmt.Printf(" : during the development phase by specifying the origin\n")
	fmt.Printf(" : of both front-end and back-end servers or external API\n")
	fmt.Printf(" : servers as the same place.\n")

	fmt.Printf("\n[Usage]\n")
	fmt.Printf(" > dev-proxy\n")
	fmt.Printf(" > dev-proxy -addr localhost:8000\n")
	fmt.Printf(" > dev-proxy -front http://localhost:3000 -back http://localhost:4000\n")
	fmt.Printf(" > dev-proxy -addr localhost:8000 -front http://localhost:3000 -back http://localhost:4000\n")
	fmt.Printf(" > dev-proxy -front [server1] -back [server2] -api [server3]\n")

	os.Exit(1)
}

func parseFlags() {
	lenArgs := len(os.Args)
	if lenArgs != 1 && lenArgs % 2 == 0 {
		help()
	}

	Flags = make(map[string]string)
	Flags["addr"] = "localhost:8000"

	fmt.Printf("Dev Proxy Server\n")

	for i, val := range os.Args {
		if i % 2 == 0 {
			continue
		} else if val[0] != '-' {
			help()
		}

		key := strings.ToLower(val[1:])

		if key == "h" || key == "help" {
			help()
		}

		arg := os.Args[i + 1]

		if key == "addr" {
			Addr = arg
		} else if key == "favicon" {
			App.Use(favicon.New(favicon.Config{ File: arg }))
		} else if arg[:4] == "http" {
			fmt.Printf(" [Proxy ] SERVER -> %s -> %s\n", key, arg)
			Flags[key] = arg
		} else {
			fmt.Printf(" [Static] SERVER -> %s -> %s\n", key, arg)
			App.Static(key, arg)
		}
	}

	fmt.Printf(" [Listen] %s -> SERVER\n", Addr)
}

func hello(c *fiber.Ctx) error {
	return c.SendString("DEV PROXY SERVER")
}

func proxyAnother(c *fiber.Ctx) error {
	key := strings.ToLower(c.Params("key"))
	url := c.Params("*")
	proxyAddr, check := Flags[key]

	if !check {
		return c.SendStatus(fiber.StatusNotFound)
	}

	if err := proxy.Do(c, proxyAddr + "/" + url); err != nil {
		return err
	}

	c.Response().Header.Del(fiber.HeaderServer)
	return nil
}

func main() {
	// Create Fiber App
	App = fiber.New()

	// Add middlewares
	App.Use(logger.New())

	// Parsing command line flags
	parseFlags()

	// Add routes
	App.All("/", hello)
	App.All("/:key/*", proxyAnother)

	// Start server
	log.Fatal(App.Listen(Addr))
}