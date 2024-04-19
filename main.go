package main

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
	"net/url"
	"os"
	"path/filepath"

	fasthttpWebsocket "github.com/fasthttp/websocket"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/kirsle/configdir"
	"github.com/yejun614/dev-proxy/data"
)

const (
	VERSION = "v1.0"
)

var (
	DB     *data.Data[ProxyData]
	App    *fiber.App
	store  = session.New()
	wsFunc = websocket.New(websocketHandler)
	wsChan = make(chan string, 10)
)

type ProxyData struct {
	Addr    string            `json:"addr" xml:"addr" form:"addr"`
	Proxies map[string]string `json:"proxies" xml:"proxies" form:"proxies"`
	Statics map[string]string `json:"statics" xml:"statics" form:"statics"`
}

func addStatics() {
	for key := range DB.Data.Statics {
		value := DB.Data.Statics[key]
		if _, err := os.Stat(value); os.IsNotExist(err) {
			log.Printf("%s: %s directory not found\n", key, value)
			continue
		}
		App.Static(key, value)
	}
}

func hello(c *fiber.Ctx) error {
	return c.SendString("DEV PROXY SERVER")
}

func sessionRedirect(c *fiber.Ctx) error {
	// session
	sess, err := store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	sessKey := sess.Get("key")
	// get key and url
	key := c.Params("key")
	log.Printf("key: %s\n", key)
	// check proxies
	_, check := DB.Data.Proxies[key]
	if !check {
		_, check = DB.Data.Statics[key]
	}
	if check {
		if key != "favicon.ico" {
			// set session
			sess.Set("key", key)
			log.Printf("session save: %s\n", key)
		}
	} else if sessKey != nil {
		// redirect
		log.Printf("session redirect: %s\n", c.Path())
		if sessKey == "" {
			c.Path(fmt.Sprintf("/%s", c.Path()[1:]))
		} else {
			c.Path(fmt.Sprintf("/%s/%s", sessKey, c.Path()[1:]))
		}
	}
	// session save
	if err := sess.Save(); err != nil {
		log.Fatal(err)
	}
	// next
	return c.Next()
}

func proxyAnother(c *fiber.Ctx) error {
	// get key and url
	key := c.Params("key")
	// check proxies
	proxyAddr, check := DB.Data.Proxies[key]
	if !check {
		key = ""
		proxyAddr, check = DB.Data.Proxies[key]

		if !check {
			// not found error
			return c.SendStatus(fiber.StatusNotFound)
		}
	}
	// proxy url
	proxyUrl, err := url.JoinPath(proxyAddr, c.Path()[len(key)+1:])
	if err != nil {
		log.Fatal(err)
	}
	// websocket
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		log.Println("websocket start")
		log.Printf("chan send %s\n", proxyUrl)
		wsChan <- proxyUrl
		return wsFunc(c)
	}
	// create proxy url
	log.Printf("Proxy: %s -> %s\n", c.Path(), proxyUrl)
	client := &fasthttp.Client{
		ReadBufferSize: 40890,
	}
	if err := proxy.DoRedirects(c, proxyUrl, 100, client); err != nil {
		return err
	}
	// set session
	sess, err := store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	sess.Set("key", key)
	if err := sess.Save(); err != nil {
		log.Fatal(err)
	}
	// done
	return nil
}

func websocketHandler(c *websocket.Conn) {
	var (
		closeChan = make(chan bool)
		client    *fasthttpWebsocket.Conn
	)

	proxyUrl, err := url.Parse(<-wsChan)
	if err != nil {
		log.Println(err)
		return
	}
	proxyUrl.Scheme = "ws"
	log.Printf("ws proxyUrl: %s\n", proxyUrl.String())
	client, _, err = fasthttpWebsocket.DefaultDialer.Dial(proxyUrl.String(), nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close()

	go func() {
		var (
			mt  int
			msg []byte
			err error
		)
		for {
			// client -> proxy
			if mt, msg, err = c.ReadMessage(); err != nil {
				log.Println(err)
				break
			}
			// debug
			log.Printf("client -> proxy: %s\n", string(msg))
			// proxy -> server
			if err = client.WriteMessage(mt, msg); err != nil {
				log.Println(err)
				break
			}
		}
		closeChan <- true
		log.Println("ws closed from client")
	}()

	go func() {
		var (
			mt  int
			msg []byte
			err error
		)
		for {
			// proxy <- server
			if mt, msg, err = client.ReadMessage(); err != nil {
				log.Println(err)
				break
			}
			// debug
			log.Printf("client <- proxy: %s\n", string(msg))
			// client <- proxy
			if err = c.WriteMessage(mt, msg); err != nil {
				log.Println(err)
				break
			}
		}
		closeChan <- true
		log.Println("ws closed from server")
	}()

	<-closeChan
	log.Println("websocket done")
}

func StartServer() {
	// config path
	configFile := "./proxy.dev.json"
	configPath := configdir.LocalConfig("dev-proxy")
	err := configdir.MakePath(configPath)
	if err == nil {
		configFile = filepath.Join(configPath, "./proxy.dev.json")
	}
	log.Printf("config file: %s\n", configFile)
	// database
	DB = data.New[ProxyData](configFile, ProxyData{
		Addr: "localhost:8000",
	})
	// create fiber app
	App = fiber.New()
	// add middlewares
	App.Use(logger.New())
	App.Use(cors.New())
	// admin routes
	admin := App.Group("/dev-proxy/")
	admin.Get("/hello", hello)
	admin.Get("/admin", adminGetPage)
	admin.Get("/data", adminGetData)
	admin.Post("/data", adminPostData)
	// session redirect middleware
	App.Use("/:key/*", sessionRedirect)
	// add statics
	addStatics()
	// add routes
	App.All("/:key/*", proxyAnother)
	// root routes
	App.All("/", proxyAnother)
	// start fiber app
	log.Println("Press Ctrl+C to shut down the server.")
	App.Listen(DB.Data.Addr)
}

func main() {
	for {
		log.Println("Server Start")
		StartServer()
	}
}
