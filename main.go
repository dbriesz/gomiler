/*
Copyright 2017 - The GoMiler Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	github "go.okkur.org/gomiler/github"
	gitlab "go.okkur.org/gomiler/gitlab"
	"go.okkur.org/gomiler/utils"
)

// Initialization of logging variable
var logger *log.Logger

// LoggerSetup Initialization
func LoggerSetup(info io.Writer) {
	logger = log.New(info, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func checkAPI(baseURL string, token string, namespace string, project string) (string, error) {
	gitlabURL := baseURL + "/api/v4/version"
	githubURL := baseURL + "/repos/" + namespace + "/" + project
	m := map[string]string{
		"gitlab": gitlabURL,
		"github": githubURL,
	}
	var resp *http.Response
	var client http.Client
	for k, v := range m {
		req, err := http.NewRequest("GET", v, nil)
		if err != nil {
			return "", err
		}
		switch k {
		case "gitlab":
			req.Header.Add("PRIVATE-TOKEN", token)
		case "github":
			req.Header.Add("Accept", "application/vnd.github.v3+json")
			req.Header.Add("Authorization", "token "+token)
		}
		resp, err = client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			return k, nil
		}
		if resp.StatusCode == 403 {
			return "", errors.New("Provided token is invalid. Access Denied.")
		}
	}
	// Check for 404 error returned from GitHub API if project not found.
	// GitLab uses the API version page as a check, so a project not found error is returned in GetProjectID instead.
	if resp.StatusCode == 404 {
		return "", fmt.Errorf("project %s not found", project)
	}
	return "", fmt.Errorf("Error: could not access GitLab or GitHub APIs")
}

func validateBaseURLScheme(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	u.Scheme = "https"
	q := u.Query()
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func main() {
	// Declaring variables for flags
	var token, baseURL, namespace, project, interval string
	var advance int
	// Command Line Parsing Starts
	flag.StringVar(&token, "token", "jGWPwqQUuf37b", "GitLab or GitHub API key/token")
	flag.StringVar(&interval, "interval", "daily", "Set milestone to daily, weekly or monthly")
	flag.StringVar(&baseURL, "url", "dev.example.com", "GitLab or GitHub API base URL")
	flag.StringVar(&namespace, "namespace", "someNamespace", "Namespace to use in GitLab or GitHub")
	flag.StringVar(&project, "project", "someProject", "Project to use in GitLab or GitHub")
	flag.IntVar(&advance, "advance", 30, "Define timeframe to generate milestones in advance")
	flag.Parse() //Command Line Parsing Ends

	// Initializing logger
	LoggerSetup(os.Stdout)

	// Validate baseURL scheme
	URL, err := validateBaseURLScheme(baseURL)
	if err != nil {
		logger.Println(err)
	}

	// Check which API to use
	api, err := checkAPI(URL, token, namespace, project)
	if err != nil {
		logger.Fatal(err)
	}

	// Calling getProjectID
	var newBaseURL, projectID string

	interval = strings.ToLower(interval)
	switch api {
	case "gitlab":
		milestoneData, err := utils.CreateMilestoneData(advance, interval, logger, api)
		if err != nil {
			logger.Fatal(err)
		}
		newBaseURL = URL + "/api/v4"
		projectID, err = gitlab.GetProjectID(newBaseURL, token, project, namespace)
		if err != nil {
			logger.Fatal(err)
		}
		err = gitlab.CreateAndDisplayNewMilestones(newBaseURL, token, projectID, milestoneData, logger)
		if err != nil {
			logger.Println(err)
		}
		closedMilestones, err := gitlab.GetClosedMilestones(newBaseURL, token, projectID, milestoneData)
		if err != nil {
			logger.Println(err)
		}
		_, err = gitlab.ReactivateClosedMilestones(closedMilestones, newBaseURL, token, projectID, logger)
		if err != nil {
			logger.Println(err)
		}
	case "github":
		milestoneData, err := utils.CreateMilestoneData(advance, interval, logger, api)
		if err != nil {
			logger.Fatal(err)
		}
		newBaseURL = URL + "/repos/" + namespace + "/"
		err = github.CreateAndDisplayNewMilestones(newBaseURL, token, project, milestoneData, logger)
		if err != nil {
			logger.Println(err)
		}
		closedMilestones, err := github.GetClosedMilestones(newBaseURL, token, project, milestoneData)
		if err != nil {
			logger.Println(err)
		}
		_, err = github.ReactivateClosedMilestones(closedMilestones, newBaseURL, token, project)
		if err != nil {
			logger.Println(err)
		}
	}
}
