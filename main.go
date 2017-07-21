package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"sort"

	"github.com/gorilla/mux"
	"github.com/mkobaly/hipchatBot/config"
	"github.com/mkobaly/hipchatBot/teamcity"
	"github.com/mkobaly/hipchatBot/util"
	"github.com/tbruyelle/hipchat-go/hipchat"
)

type HipChatBasicMessage struct {
	Color         string `json:"color"`
	Message       string `json:"message"`
	Notify        bool   `json:"notify"`
	MessageFormat string `json:"message_format"`
}

type HipchatWebhook struct {
	Event         string
	Item          HipchatItem
	OauthClientID string `json:"oauth_client_id"`
	WebhookID     int    `json:"webhook_id"`
}

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
	rooms   map[string]*RoomConfig
	builder *teamcity.Builder
	cfg     *config.Config
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
	var p HipchatWebhook
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.Fatalf("Err decoding request body: %v\n", err)
	}

	cmd := strings.Split(p.Item.Message.Message, " ")
	if len(cmd) < 2 {
		postToHipchat(c.cfg.HipchatURL, parseHTMLTemplate("help", nil), "yellow", "html")
		return
	}
	action := cmd[1]

	switch action {
	case "list":
		builds, err := c.builder.GetBuilds()
		sort.Sort(teamcity.ById(builds))
		if err == nil {
			var bIds []string
			for _, r := range builds {
				if strings.HasSuffix(r.ID, "_RC") || strings.HasSuffix(r.ID, "_CI") {
					bIds = append(bIds, r.ID)
				}
			}
			message := parseHTMLTemplate("list", bIds)
			postToHipchat(c.cfg.HipchatURL, message, "green", "html")
		} else {
			postToHipchat(c.cfg.HipchatURL, "<b>Error getting build list</b>", "red", "html")
		}
		return
	case "kick":
		if len(cmd) != 4 {
			postToHipchat(c.cfg.HipchatURL, parseHTMLTemplate("help", nil), "yellow", "html")
			return
		}
		buildConfig, branch := cmd[2], cmd[3]
		bi := teamcity.BuildInfo{BuildConfigID: buildConfig, Branch: branch}
		c.builder.SetBuildInfo(bi)

		params := make(map[string]string)
		params["Branch"] = branch

		if err := c.builder.Build(params); err != nil {
			postToHipchat(c.cfg.HipchatURL, "<b>Error kicking off build</b>", "red", "html")
		} else {

			data := struct {
				BuildConfigID string
				Branch        string
				TaskID        string
			}{
				BuildConfigID: buildConfig,
				Branch:        branch,
				TaskID:        strings.Split(c.builder.BuildResult.HREF, ":")[1],
			}
			message := parseHTMLTemplate("kick", data)
			postToHipchat(c.cfg.HipchatURL, message, "green", "html")
		}
		return
	case "status":
		if len(cmd) != 3 {
			postToHipchat(c.cfg.HipchatURL, parseHTMLTemplate("help", nil), "yellow", "html")
			return
		}
		taskId := cmd[2]
		b, err := c.builder.GetBuildStatus1(taskId)
		if err != nil {
			fmt.Println(err.Error())
		}
		color := "yellow"
		if b.State == "finished" && b.Status == "SUCCESS" {
			color = "green"
		}
		if b.State == "finished" && b.Status == "FAILURE" {
			color = "red"
		}
		message := parseHTMLTemplate("status", b)
		postToHipchat(c.cfg.HipchatURL, message, color, "html")

		return
	case "--help":
		postToHipchat(c.cfg.HipchatURL, parseHTMLTemplate("help", nil), "green", "html")
		return
	default:
		postToHipchat(c.cfg.HipchatURL, parseHTMLTemplate("help", nil), "yellow", "html")
	}
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

func watchForFinishedBuild(b *teamcity.Builder, hipchatURL string) error {
	for {
		time.Sleep(time.Second * 2)
		fmt.Println("In WatchForFinishedBuild..2 sec delay")
		br := b.BuildResult
		fmt.Printf("Build Result Id: %v State: %v", br.BuildTypeID, br.State)
		err := b.GetBuildStatus(br)
		fmt.Printf("Build Result Id: %v State: %v", br.BuildTypeID, br.State)
		if err != nil {
			return err
		}
		if br.State == "finished" {
			color := "red"
			if br.Status == "SUCCESS" {
				color = "green"
			}
			message := "Build complete for " + br.BuildTypeID + " State: " + br.State + " Status: " + br.Status
			postToHipchat(hipchatURL, message, color, "text")
			return nil
		}
	}
}

func parseHTMLTemplate(templateName string, data interface{}) string {
	var buffer bytes.Buffer
	t := template.New(templateName + ".html")
	var err error
	tb, err := Asset("templates/" + templateName + ".html")
	s := string(tb)
	t, err = t.Parse(s)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = t.Execute(&buffer, data)
	if err != nil {
		fmt.Println(err.Error())
	}
	return buffer.String()
}

func postToHipchat(hipchatURL string, message string, color string, format string) error {
	m := HipChatBasicMessage{Color: color, Notify: false, MessageFormat: format, Message: message}
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(m)
	resp, err := http.Post(hipchatURL, "application/json", b)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return errors.New("Non 200 response status")
	}
	return err
}

// BASE_URL=https://6011fb9f.ngrok.io ./hipchatBot
func main() {
	var (
		static = flag.String("static", "./static/", "static folder")
	)
	flag.Parse()

	config := config.NewConfig("config.yaml")

	c := &Context{
		baseURL: config.NgrokURL,
		static:  *static,
		rooms:   make(map[string]*RoomConfig),
		builder: teamcity.New(config.Teamcity),
		cfg:     config,
	}

	log.Printf("Base HipChat integration v0.10 - running on port:%v", config.Port)

	r := c.routes()
	http.Handle("/", r)
	http.ListenAndServe(":"+strconv.Itoa(config.Port), nil)
}
