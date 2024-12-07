package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var gatewayHost string = os.Getenv("GATEWAY_HOST")

type Page struct {
	Title         string
	Body          []byte
	Authenticated bool
	UserId        string
	UserName      string
}

func (p *Page) checkUser(c *gin.Context) {
	url := "http://" + gatewayHost + "/user/active"
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		p.Authenticated = false
		return
	}
	cookie, err := c.Request.Cookie("auth")
	if err != nil {
		p.Authenticated = false
		return
	}
	req.AddCookie(cookie)
	response, err := client.Do(req)
	if err != nil {
		p.Authenticated = false
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		p.Authenticated = false
		return
	}
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		p.Authenticated = false
		return
	}
	var returnData map[string]string
	if err := json.Unmarshal(responseData, &returnData); err != nil {
		p.Authenticated = false
		return
	}
	if returnData["message"] != "" {
		p.Authenticated = false
		return
	}
	p.Authenticated = true
	p.UserId = returnData["id"]
	p.UserName = returnData["name"]
}

func loadPage(title string) (*Page, error) {
	filename := "/app/pages/" + title + ".html"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func getIndex(c *gin.Context) {
	p, err := loadPage("index")
	p.checkUser(c)
	fmt.Println(p.Authenticated)
	if err != nil {
		return
	}
	renderTemplate(c.Writer, "default", p)
}

func getRegister(c *gin.Context) {
	p, err := loadPage("register")
	p.checkUser(c)
	if err != nil {
		return
	}
	renderTemplate(c.Writer, "default", p)
}

func postRegister(c *gin.Context) {
	data := map[string]string{"name": c.Request.FormValue("name")}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	url := "http://" + gatewayHost + "/user/register"
	response, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}
	var returnData map[string]string
	if err := json.Unmarshal(responseData, &returnData); err != nil {
		return
	}
	c.Redirect(http.StatusFound, "/login")
}

func getLogin(c *gin.Context) {
	p, err := loadPage("login")
	if err != nil {
		return
	}
	p.checkUser(c)
	if p.Authenticated {
		c.Redirect(http.StatusFound, "/user/"+p.UserId)
		return
	}
	renderTemplate(c.Writer, "default", p)
}

func getLogout(c *gin.Context) {
	c.SetCookie("auth", "", -1, "/", "localhost", false, true)
	c.Redirect(http.StatusFound, "/")
}

func postLogin(c *gin.Context) {
	name := c.Request.FormValue("name")
	// TODO: check a to be implemented password here
	url := "http://" + gatewayHost + "/user/login"
	data := map[string]string{
		"name": name,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}
	body := bytes.NewBuffer(jsonData)
	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	defer resp.Body.Close()
	for _, respCookie := range resp.Cookies() {
		c.SetCookie(respCookie.Name, respCookie.Value, int(respCookie.MaxAge), respCookie.Path, respCookie.Domain, respCookie.Secure, respCookie.HttpOnly)
	}
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response data"})
		return
	}
	var userJson map[string]string
	if err := json.Unmarshal(respData, &userJson); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse json data"})
		return
	}
	c.Redirect(http.StatusFound, "/user/"+userJson["id"])
}

func getUserPage(c *gin.Context) {
	id := c.Param("id")
	url := "http://" + gatewayHost + "/user/id/" + id
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	cookie, err := c.Request.Cookie("auth")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	req.AddCookie(cookie)
	response, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get response"})
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "statuscode error"})
		return
	}
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Responsedata error"})
		return
	}
	var returnData map[string]string
	if err := json.Unmarshal(responseData, &returnData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Json unmarshal error"})
		return
	}
	p := Page{Title: returnData["name"], Body: []byte("<a href='/guestbook/create'>Create guestbook.</a>")}
	p.Authenticated = true
	p.UserId = id
	p.UserName = returnData["name"]
	renderTemplate(c.Writer, "default", &p)
}

func getCreateGuestbook(c *gin.Context) {
	p, err := loadPage("create_guestbook")
	if err != nil {
		return
	}
	p.Title = "Create New Guestbook"
	p.checkUser(c)
	renderTemplate(c.Writer, "default", p)
}

func postCreateGuestbook(c *gin.Context) {
	data := map[string]interface{}{
		"domain":          c.Request.FormValue("domain"),
		"requireApproval": (c.Request.FormValue("approval") == "on"),
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request"})
		return
	}
	url := "http://" + gatewayHost + "/guestbook/new"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Request creation failed"})
		return
	}
	client := http.Client{}
	cookie, err := c.Request.Cookie("auth")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read cookie"})
		return
	}
	req.AddCookie(cookie)
	response, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to post request"})
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}
	var returnData map[string]interface{}
	if err := json.Unmarshal(responseData, &returnData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response data"})
		return
	}
	c.Redirect(http.StatusFound, "/guestbook/"+returnData["id"].(string))
}

func getGuestbook(c *gin.Context) {
	url := "http://" + gatewayHost + "/guestbook/" + c.Param("id")
	cookie, err := c.Cookie("auth")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read cookie"})
		return
	}
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("auth", cookie)
	response, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed"})
		return
	}
	defer response.Body.Close()
	var guestbookData map[string]interface{}
	responseData, err := io.ReadAll(response.Body)
	fmt.Println(response.StatusCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response data"})
		return
	}
	if err := json.Unmarshal(responseData, &guestbookData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parses guestbook data"})
		return
	}
	p := Page{Title: guestbookData["domain"].(string), Body: bytes.NewBufferString(guestbookData["ownerId"].(string)).Bytes()}
	p.checkUser(c)
	renderTemplate(c.Writer, "default", &p)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	t, _ := template.ParseFiles("/app/templates/" + tmpl + ".gohtml")
	t.Execute(w, p)
}

func main() {
	hostname := os.Getenv("GUESTBOOK_ROOT_DOMAIN")

	router := gin.Default()

	strictCors := cors.New(cors.Config{
		AllowOrigins:     []string{hostname},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type", "Origin"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})

	rootGroup := router.Group("/")
	rootGroup.Use(strictCors)
	{
		rootGroup.GET("/", getIndex)
		rootGroup.GET("/register", getRegister)
		rootGroup.POST("/register", postRegister)
		rootGroup.GET("/login", getLogin)
		rootGroup.POST("/login", postLogin)
		rootGroup.GET("/logout", getLogout)
		rootGroup.GET("/user/:id", getUserPage)
		rootGroup.GET("/guestbook/create", getCreateGuestbook)
		rootGroup.POST("/guestbook/create", postCreateGuestbook)
		rootGroup.GET("/guestbook/:id", getGuestbook)
	}

	router.Run()
}
