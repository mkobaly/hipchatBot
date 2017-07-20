package teamcity

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mkobaly/hipchatBot/config"
	"github.com/mkobaly/teamcity"
)

//BuildInfo represents a Build Configuration and branch in Teamcity
type BuildInfo struct {
	BuildConfigID string
	Branch        string
}

type IBuilder interface {
	//Build(id string, branch string) error
	Build() error
	Done() <-chan struct{}
	Result() interface{}
}

type Builder struct {
	Credentials config.UserCredential
	BuildInfo   BuildInfo
	// buildID     string
	// branch      string
	client      *teamcity.Client
	BuildResult *teamcity.Build
}

type ById []*teamcity.BuildType

func (a ById) Len() int           { return len(a) }
func (a ById) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ById) Less(i, j int) bool { return a[i].ID < a[j].ID }

//New will create a new teamcity Builder
func New(creds config.UserCredential) *Builder {
	var b = new(Builder)
	b.Credentials = creds
	b.client = teamcity.New(creds.URL, creds.Username, creds.Password)
	return b
}

func (b *Builder) SetBuildInfo(bi BuildInfo) error {
	b.BuildInfo = bi
	b.BuildResult = new(teamcity.Build)
	return nil
}

//Build will kick off a TeamCity build
func (b *Builder) Build(params map[string]string) error {
	if (BuildInfo{}) == b.BuildInfo {
		return errors.New("Build Info not set yet so unable to build")
	}

	//client := teamcity.New(b.Credentials.URL, b.Credentials.Username, b.Credentials.Password)
	x, err := b.client.QueueBuild(b.BuildInfo.BuildConfigID, b.BuildInfo.Branch, params)
	if err != nil {
		return err
	}
	b.BuildResult = x
	return nil
}

func (b *Builder) BuildResultToJson() string {
	r, _ := json.MarshalIndent(b.BuildResult, "", "\t")
	return string(r)
}

// //GetBuild will return the current state of the build
func (b *Builder) GetBuild() error {
	br, err := b.client.GetBuild(strconv.FormatInt(b.BuildResult.ID, 10))
	b.BuildResult = br
	return err
}

//VerifyBuildStatus will return the current state of the build
func (b *Builder) VerifyBuildStatus() error {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", b.Credentials.URL+b.BuildResult.HREF, nil)
	req.SetBasicAuth(b.Credentials.Username, b.Credentials.Password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		panic(err.Error())
	}
	if err := json.NewDecoder(resp.Body).Decode(b.BuildResult); err != nil {
		return err
	}
	return nil
}

func (b *Builder) GetBuildStatus1(taskId string) (teamcity.Build, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", b.Credentials.URL+"httpAuth/app/rest/buildQueue/taskId:"+taskId, nil)
	req.SetBasicAuth(b.Credentials.Username, b.Credentials.Password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		panic(err.Error())
	}
	var br teamcity.Build
	err = json.NewDecoder(resp.Body).Decode(&br)
	return br, err

}

func (b *Builder) GetBuildStatus(br *teamcity.Build) error {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", b.Credentials.URL+br.HREF, nil)
	req.SetBasicAuth(b.Credentials.Username, b.Credentials.Password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		panic(err.Error())
	}
	if err := json.NewDecoder(resp.Body).Decode(br); err != nil {
		return err
	}
	return nil
}

//GetArtifactVersion will return the version number of the build artifact
func (b *Builder) GetArtifactVersion() (string, error) {
	client := teamcity.New(b.Credentials.URL, b.Credentials.Username, b.Credentials.Password)
	version, err := client.GetArtifact(b.BuildResult.ID)
	if err != nil {
		var s = strings.Replace(version.Name, ".zip", "", 1)
		var parts = strings.Split(s, ".v")
		if len(parts) == 2 {
			return parts[1], nil
		}
	}
	return "", err
}

//GetArtifactVersionByID will return the version number of the build artifact
func (b *Builder) GetArtifactVersionByID(id int64) (string, error) {
	client := teamcity.New(b.Credentials.URL, b.Credentials.Username, b.Credentials.Password)
	version, err := client.GetArtifact(id)
	if err != nil {
		var s = strings.Replace(version.Name, ".zip", "", 1)
		var parts = strings.Split(s, ".v")
		if len(parts) == 2 {
			return parts[1], nil
		}
	}
	return "", err
}

//GetBuilds will list out all available builds on TeamCity
func (b *Builder) GetBuilds() ([]*teamcity.BuildType, error) {
	//client := teamcity.New(b.Credentials.URL, b.Credentials.Username, b.Credentials.Password)
	builds, err := b.client.GetBuildTypes()
	return builds, err
}

//VerifyBuildStatus will return the current state of the build
func (b *Builder) GetLastestBuild(buildType string) (string, error) {
	b.BuildResult = new(teamcity.Build)
	client := &http.Client{}
	var url = fmt.Sprintf("%s/httpAuth/app/rest/buildTypes/id:%s/builds/running:false,status:success", b.Credentials.URL, buildType)
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(b.Credentials.Username, b.Credentials.Password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		panic(err.Error())
	}
	if err := json.NewDecoder(resp.Body).Decode(b.BuildResult); err != nil {
		return "", err
	}
	ff, err := b.GetArtifactVersion()
	return ff, err
	//return b.GetArtifactVersionByID(b.BuildResult.ID)
}
