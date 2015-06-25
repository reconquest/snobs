package main

import (
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/bndr/gopencils"
	"github.com/docopt/docopt-go"
	"github.com/zazab/zhash"
)

const (
	usage = `Snobs 1.0

Usage:
    snobs [options]

Options:
    -c <config>   use specified configuration file
                  [default: /etc/snobs/snobs.conf].
`
)

var (
	reStashURL = regexp.MustCompile(
		`(https?://.*/)` +
			`(users|projects)/([^/]+)` +
			`/repos/([^/]+)` +
			`/pull-requests/(\d+)`)
)

type SnobServer struct {
	config zhash.Hash
	api    *gopencils.Resource
}

type ResponseUsers struct {
	Users []struct {
		Name string `json:"name"`
	} `json:"values"`
}

type ResponsePullRequest struct {
	Version float64 `json:"version"`
	Author  struct {
		User struct {
			Name string `json:"name"`
		} `json:"user"`
	} `json:"author"`
}

func main() {
	args, err := docopt.Parse(usage, nil, true, "blah", false, true)
	if err != nil {
		log.Fatal(err)
	}

	var (
		configPath = args["-c"].(string)
	)

	config, err := getConfig(configPath)
	if err != nil {
		log.Fatalf("can't load config: %s", err.Error())
	}

	server, err := NewSnobServer(config)
	if err != nil {
		log.Fatal(err)
	}

	err = server.ListenHTTP()
	if err != nil {
		log.Fatal(err)
	}
}

func NewSnobServer(config zhash.Hash) (*SnobServer, error) {
	server := &SnobServer{}

	err := server.SetConfig(config)
	if err != nil {
		return nil, err
	}

	var (
		stashHost, _ = server.config.GetString("stash")
		stashUser, _ = server.config.GetString("user")
		stashPass, _ = server.config.GetString("pass")
	)

	server.api = gopencils.Api(
		"http://"+stashHost+"/rest/api/1.0",
		&gopencils.BasicAuth{stashUser, stashPass},
	)

	return server, nil
}

func (server *SnobServer) SetConfig(config zhash.Hash) error {
	params := []string{
		"listen", "stash", "user", "pass",
	}

	for _, paramName := range params {
		_, err := config.GetString(paramName)
		if err != nil {
			return err
		}
	}

	server.config = config

	return nil
}

func (server *SnobServer) ListenHTTP() error {
	address, _ := server.config.GetString("listen")

	httpServer := &http.Server{
		Addr:    address,
		Handler: server,
	}

	return httpServer.ListenAndServe()
}

func (server *SnobServer) ServeHTTP(
	response http.ResponseWriter, request *http.Request,
) {
	uriParts := strings.SplitN(
		strings.Trim(request.URL.Path, "/"),
		"/", 2,
	)
	if len(uriParts) < 2 {
		http.Error(response, "%group%/%pull-request%", http.StatusBadRequest)
		return
	}

	var (
		usersGroup     = uriParts[0]
		pullRequestURL = uriParts[1]
	)

	users, err := server.GetUsers(usersGroup)
	if err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	matches := reStashURL.FindStringSubmatch(pullRequestURL)
	if len(matches) == 0 {
		http.Error(response, "wrong url", http.StatusBadRequest)
		return
	}

	var (
		project     = matches[3]
		repository  = matches[4]
		pullRequest = matches[5]
	)

	err = server.AddReviewers(project, repository, pullRequest, users)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Error(response, "reviewers added", http.StatusOK)
}

func (server *SnobServer) GetUsers(group string) ([]string, error) {
	request, err := server.api.Res(
		"admin/groups/more-members", &ResponseUsers{},
	).Get(map[string]string{"context": group})

	if err != nil {
		return []string{}, nil
	}

	response := request.Response.(*ResponseUsers)
	names := []string{}
	for _, user := range response.Users {
		names = append(names, user.Name)
	}

	return names, nil
}

func (server *SnobServer) AddReviewers(
	project string, repository string, pullRequest string,
	users []string,
) error {
	author, version, err := server.GetPullRequestInfo(
		project, repository, pullRequest,
	)
	if err != nil {
		return err
	}

	stashUser, _ := server.config.GetString("user")
	reviewers := getReviewers(
		users, []string{author, stashUser},
	)

	payload := map[string]interface{}{
		"id":        pullRequest,
		"version":   version,
		"reviewers": reviewers,
	}

	_, err = server.api.Res("projects").Res(project).
		Res("repos").Res(repository).
		Res("pull-requests").Res(pullRequest, &map[string]interface{}{}).
		Put(payload)

	return err
}

func (server *SnobServer) GetPullRequestInfo(
	project string, repository string, pullRequest string,
) (string, int64, error) {
	request, err := server.api.Res("projects").Res(project).
		Res("repos").Res(repository).
		Res("pull-requests").Res(pullRequest, &ResponsePullRequest{}).
		Get()

	if err != nil {
		return "", 0, err
	}

	info := *request.Response.(*ResponsePullRequest)

	return info.Author.User.Name, int64(info.Version), nil
}

func getConfig(path string) (zhash.Hash, error) {
	var configData map[string]interface{}

	_, err := toml.DecodeFile(path, &configData)
	if err != nil {
		return zhash.Hash{}, err
	}

	return zhash.HashFromMap(configData), nil
}

func getReviewers(users []string, ignoreUsers []string) []map[string]interface{} {
	reviewers := []map[string]interface{}{}
	for _, user := range users {
		ignore := false
		for _, ignoreUser := range ignoreUsers {
			if ignoreUser == user {
				ignore = true
				break
			}
		}

		if ignore {
			continue
		}

		reviewers = append(reviewers, map[string]interface{}{
			"user": map[string]interface{}{
				"name": user,
			},
		})
	}

	return reviewers
}
