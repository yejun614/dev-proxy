package main

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
	"github.com/yejun614/dev-proxy/data"
)

//go:embed web/index.html
var adminHTML []byte

func adminGetPage(c *fiber.Ctx) error {
	return c.Format(adminHTML)
}

func adminGetData(c *fiber.Ctx) error {
	return c.JSON(DB)
}

func adminPostData(c *fiber.Ctx) error {
	// body parser
	body := new(data.Data[ProxyData])
	if err := c.BodyParser(body); err != nil {
		return err
	}
	// filename
	DB.Filepath = body.Filepath
	// data
	DB.Data = body.Data
	// save
	DB.Save()
	// shutdown
	go func() { App.Shutdown() }()
	// redirect
	return c.Status(fiber.StatusOK).Redirect("/dev-proxy/admin")
}
