package main

import (
	"context"
	"embed"
	_ "embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed view/*.html
var views embed.FS

func main() {
	if os.Getenv("PASSWORD") == "" {
		log.Fatal("PASSWORD environment variable not set")
		return
	}
	if os.Getenv("CLUSTER_MANAGER_ADDRESS") == "" {
		log.Fatal("CLUSTER_MANAGER_ADDRESS environment variable not set")
		return
	}
	if os.Getenv("ATOMIX_IMAGE") == "" {
		log.Fatal("ATOMIX_IMAGE environment variable not set")
		return
	}
	if os.Getenv("ONOS_IMAGE") == "" {
		log.Fatal("ONOS_IMAGE environment variable not set")
		return
	}

	backupAtomixIPFromFile()
	backupOnosIPFromFile()
	go onosStatusListener()
	port := "8080"

	wg := sync.WaitGroup{}
	cli, err := client.NewClientWithOpts(client.WithHost("unix:///var/run/docker.sock"))
	if err != nil {
		log.Fatal(err)
	}
	// Initialize the Manager struct
	manager := Manager{
		ctx:    context.Background(),
		client: cli,
	}

	wg.Add(1)
	server := echo.New()
	// Instantiate a template registry and register all html files inside the view folder
	server.Renderer = &TemplateRegistry{
		templates: template.Must(template.ParseFS(views, "view/*.html")),
	}
	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(middleware.CORS())

	server.Pre(func (next echo.HandlerFunc) echo.HandlerFunc {
		return func (c echo.Context) error {
			if strings.Contains(c.Request().RequestURI, "admin") {
				cookie, err := c.Cookie("sessionID")
				if err != nil {
					return c.Redirect(http.StatusTemporaryRedirect, "/login")
				}
				if !isSessionTokenValid(cookie.Value) {
					return c.Redirect(http.StatusTemporaryRedirect, "/logout")
				}
			}
			return next(c)
		}
	})
	

	// --- CONFIG API ---

	// Fetch atomix config
	server.GET("/atomix/config", func(c echo.Context) error {
		hostname := c.Request().Header.Get("hostname")
		node, err := manager.getNodeDetailsFromTaskID(hostname)
		if err != nil {
			return c.String(500, "failed")
		}
		return c.String(200, generateAtomixConfig(node))
	})

	// Fetch onos config
	server.GET("/onos/config", func(c echo.Context) error {
		hostname := c.QueryParam("hostname")
		node, err := manager.getNodeDetailsFromTaskID(hostname)
		if err != nil {
			return c.String(500, "failed")
		}
		return c.String(200, generateOnosConfig(node))
	})

	// --- ATOMIX STATUS UPDATER ---

	// Update state of atomix node -- up
	server.GET("/atomix/up", func(c echo.Context) error {
		hostname := c.Request().Header.Get("hostname")
		node, err := manager.getNodeDetailsFromTaskID(hostname)
		if err != nil {
			return c.String(500, "failed")
		}
		upAtomix(node.IP)
		return c.String(200, "ok")
	})

	// Update state of atomix node -- down
	server.GET("/atomix/down", func(c echo.Context) error {
		hostname := c.Request().Header.Get("hostname")
		node, err := manager.getNodeDetailsFromTaskID(hostname)
		if err != nil {
			return c.String(500, "failed")
		}
		downAtomix(node.IP)
		return c.String(200, "ok")
	})

	// Get runing state of atomix process
	server.GET("/admin/atomix/state", func(c echo.Context) error {
		state := atomixStatus
		jsonData, err := json.Marshal(state)
		if err != nil {
			return c.String(500, "failed")
		}

		return c.String(200, string(jsonData))
	})

	// Get runing state of onos process
	server.GET("/admin/onos/state", func(c echo.Context) error {
		state := onosStatus
		jsonData, err := json.Marshal(state)
		if err != nil {
			return c.String(500, "failed")
		}

		return c.String(200, string(jsonData))
	})


	// --- FETCH NODES ---
	server.GET("/admin/nodes", func(c echo.Context) error {
		nodes, err := manager.availableNodes()
		if err != nil {
			return c.String(500, err.Error())
		}
		jsonData, err := json.Marshal(nodes)
		if err != nil {
			return c.String(500, "failed unexpected reason")
		}
		return c.String(200, string(jsonData))
	})

	// --- ADD NODE ---

	// Add atomix nodes ip
	server.GET("/admin/atomix/add", func(c echo.Context) error {
		if len(atomixNodesIP) > 0 {
			return c.String(400, "Atomix nodes already added ! For more nodes, need to recreate the cluster")
		}
		ips := c.QueryParam("ips")
		var ipList []string = strings.Split(ips, ",")
		err := addAtomixNodes(manager, ipList)
		if err != nil {
			return c.String(500, err.Error())
		}
		return c.String(200, "All atomix nodes added !")
	})

	// Add onos nodes ip
	server.GET("/admin/onos/add", func(c echo.Context) error {
		ips := c.QueryParam("ips")
		var ipList []string = strings.Split(ips, ",")
		err := addOnosNodes(manager, ipList)
		if err != nil {
			return c.String(500, err.Error())
		}
		return c.String(200, "onos nodes added !")
	})

	server.GET("/admin/onos/remove", func(c echo.Context) error {
		ips := c.QueryParam("ips")
		var ipList []string = strings.Split(ips, ",")
		err := removeOnosNodes(manager, ipList)
		if err != nil {
			return c.String(500, err.Error())
		}
		return c.String(200, "onos nodes removed !")
	})

	// -- SERVICE DEPLOY --

	// Deploy atomix service
	server.GET("/admin/atomix/deploy", func(c echo.Context) error {
		// Create atomix service
		err := manager.CreateAtomixService("atomix", os.Getenv("ATOMIX_IMAGE"))
		if err != nil {
			return c.String(500, "failed")
		}
		return c.String(200, "ok")
	})

	// Deploy onos service
	server.GET("/admin/onos/deploy", func(c echo.Context) error {
		// Create onos service
		err := manager.CreateOnosService("onos", os.Getenv("ONOS_IMAGE"))
		if err != nil {
			return c.String(500, "failed")
		}
		return c.String(200, "ok")
	})

	// -- SERVICE STATUS --

	// Get atomix service status
	server.GET("/admin/atomix/status", func(c echo.Context) error {
		// Create atomix service
		status := manager.ServiceExists("atomix")
		return c.String(200, strconv.FormatBool(status))
	})
	
	// Get onos service status
	server.GET("/admin/onos/status", func(c echo.Context) error {
		// Create onos service
		status := manager.ServiceExists("onos")
		return c.String(200, strconv.FormatBool(status))
	})


	// -- SERVICE DELETE --

	// Delete atomix service
	server.GET("/admin/atomix/delete", func(c echo.Context) error {
		// Delete atomix service
		err := manager.DeleteService("atomix")
		if err != nil {
			return c.String(500, "failed")
		}
		return c.String(200, "ok")
	})

	server.GET("/admin/onos/delete", func(c echo.Context) error {
		// Delete onos service
		err := manager.DeleteService("onos")
		if err != nil {
			return c.String(500, "failed")
		}
		return c.String(200, "ok")
	})

	// -- RESET SYSTEM --
	server.GET("/admin/reset", func(c echo.Context) error {
		manager.DeleteService("atomix")
		manager.DeleteService("onos")
		manager.DeleteLabelsFromAllNodes("atomix")
		manager.DeleteLabelsFromAllNodes("onos")
		atomixNodesIP = []string{}
		onosNodesIP = []string{}
		atomixStatus = make(map[string]bool, 200)
		onosStatus = make(map[string]bool, 200)
		dumpOnosIPInFile()
		dumpAtomixIPInFile()
		return c.String(200, "System reseted !")
	})

	// -- ADMIN PANEL --
	server.GET("/admin", func(c echo.Context) error {
		return c.Render(http.StatusOK, "admin.html", nil)
	})

	server.GET("/login", func(c echo.Context) error {
		return c.Render(http.StatusOK, "login.html", nil)
	})

	server.POST("/login", func(c echo.Context) error {
		password := c.FormValue("password")
		if password == os.Getenv("PASSWORD") {
			cookie := new(http.Cookie)
			cookie.Name = "sessionID"
			cookie.Value = generateSessionToken()
			cookie.Expires = time.Now().Add(1 * time.Hour)
			c.SetCookie(cookie)
			return c.Redirect(http.StatusSeeOther, "/admin")
		}
		return c.Redirect(http.StatusSeeOther, "/login")
	})

	server.GET("/logout", func(c echo.Context) error {
		cookie, err := c.Cookie("sessionID")
		if err != nil {
			deleteSessionToken(cookie.Value)
		}
		cookie = new(http.Cookie)
		cookie.Name = "sessionID"
		cookie.Value = ""
		cookie.Expires = time.Now().Add(-1 * time.Hour)
		c.SetCookie(cookie)
		return c.Redirect(http.StatusSeeOther, "/login")
	})

	// -- SPECIAL ROUTES --
	// Swarm join token
	server.POST("/swarm/join", func(c echo.Context) error {
		password := c.FormValue("password")
		if password != os.Getenv("PASSWORD") {
			return c.String(401, "Unauthorized")
		}
		token, err := manager.client.SwarmInspect(manager.ctx)
		if err != nil {
			return c.String(500, err.Error())
		}
		if token.JoinTokens.Worker == "" {
			return c.String(500, "No token available")
		}
		return c.String(200, token.JoinTokens.Worker)
	})


	// Rediretc
	server.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/admin")
	})

	err = server.Start(":" + port)
	if err != nil {
		log.Fatal(err)
	}
}
