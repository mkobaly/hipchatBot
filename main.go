package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mkobaly/hipchatBot/util"
	"github.com/tbruyelle/hipchat-go/hipchat"
)

type HipchatItem struct {
	Message *hipchat.Message
	Room    *hipchat.Room
}

// RoomConfig holds information to send messages to a specific room
type RoomConfig struct {
	token *hipchat.OAuthAccessToken
	hc    *hipchat.Client
	name  string
}

// Context keep context of the running application
type Context struct {
	baseURL string
	static  string
	//rooms per room OAuth configuration and client
	rooms map[string]*RoomConfig
}

func (c *Context) healthcheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(`{"alive": true}`)
}

func (c *Context) atlassianConnect(w http.ResponseWriter, r *http.Request) {
	lp := path.Join("./static", "atlassian-connect.json")
	vals := map[string]string{
		"LocalBaseUrl": c.baseURL,
	}
	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		log.Fatalf("%v", err)
	}
	tmpl.ExecuteTemplate(w, "config", vals)
}

func (c *Context) installable(w http.ResponseWriter, r *http.Request) {
	authPayload, err := util.DecodePostJSON(r, true)
	if err != nil {
		log.Fatalf("Parsed auth data failed:%v\n", err)
	}

	credentials := hipchat.ClientCredentials{
		ClientID:     authPayload["oauthId"].(string),
		ClientSecret: authPayload["oauthSecret"].(string),
	}
	roomName := strconv.Itoa(int(authPayload["roomId"].(float64)))
	newClient := hipchat.NewClient("")
	tok, _, err := newClient.GenerateToken(credentials, []string{hipchat.ScopeSendNotification})
	if err != nil {
		log.Fatalf("Client.GetAccessToken returns an error %v", err)
	}
	rc := &RoomConfig{
		name: roomName,
		hc:   tok.CreateClient(),
	}
	c.rooms[roomName] = rc

	util.PrintDump(w, r, false)
	json.NewEncoder(w).Encode([]string{"OK"})
}

func (c *Context) config(w http.ResponseWriter, r *http.Request) {
	signedRequest := r.URL.Query().Get("signed_request")
	lp := path.Join("./static", "layout.hbs")
	fp := path.Join("./static", "config.hbs")
	vals := map[string]string{
		"LocalBaseUrl":  c.baseURL,
		"SignedRequest": signedRequest,
		"HostScriptUrl": c.baseURL,
	}
	tmpl, err := template.ParseFiles(lp, fp)
	if err != nil {
		log.Fatalf("%v", err)
	}
	tmpl.ExecuteTemplate(w, "layout", vals)
}

func (c *Context) hook(w http.ResponseWriter, r *http.Request) {
	payLoad, err := util.DecodePostJSON(r, true)
	if err != nil {
		log.Fatalf("Parsed auth data failed:%v\n", err)
	}
	//roomID := strconv.Itoa(int((payLoad["item"].(map[string]interface{}))["room"].(map[string]interface{})["id"].(float64)))

	util.PrintDump(w, r, true)

	//log.Printf("Sending notification to %s\n", roomID)
	// notifRq := &hipchat.NotificationRequest{
	// 	Message:       "nice <strong>Happy Hook Day!</strong>",
	// 	MessageFormat: "html",
	// 	Color:         "red",
	// }
	log.Printf("payload: %v\n", payLoad)
	// if _, ok := c.rooms[roomID]; ok {
	// 	_, err = c.rooms[roomID].hc.Room.Notification(roomID, notifRq)
	// 	if err != nil {
	// 		log.Printf("Failed to notify HipChat channel:%v\n", err)
	// 	}
	// } else {
	// 	log.Printf("Room is not registered correctly:%v\n", c.rooms)
	// }
}

// routes all URL routes for app add-on
func (c *Context) routes() *mux.Router {
	r := mux.NewRouter()
	//healthcheck route required by Micros
	r.Path("/healthcheck").Methods("GET").HandlerFunc(c.healthcheck)
	//descriptor for Atlassian Connect
	r.Path("/").Methods("GET").HandlerFunc(c.atlassianConnect)
	r.Path("/atlassian-connect.json").Methods("GET").HandlerFunc(c.atlassianConnect)

	// HipChat specific API routes
	r.Path("/installable").Methods("POST").HandlerFunc(c.installable)
	r.Path("/config").Methods("GET").HandlerFunc(c.config)
	r.Path("/hook").Methods("POST").HandlerFunc(c.hook)

	r.PathPrefix("/").Handler(http.FileServer(http.Dir(c.static)))
	return r
}

// BASE_URL=https://6011fb9f.ngrok.io ./hipchatBot
func main() {
	var (
		port    = flag.String("port", "8080", "web server port")
		static  = flag.String("static", "./static/", "static folder")
		baseURL = flag.String("baseurl", os.Getenv("BASE_URL"), "local base url")
	)
	flag.Parse()

	c := &Context{
		baseURL: *baseURL,
		static:  *static,
		rooms:   make(map[string]*RoomConfig),
	}

	log.Printf("Base HipChat integration v0.10 - running on port:%v", *port)

	r := c.routes()
	http.Handle("/", r)
	http.ListenAndServe(":"+*port, nil)
}
