package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/pubsub/v1"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

var (
	apiTokens []string
)

func init() {
	apiTokens = nil
	http.HandleFunc("/", postHandler)
}

func mustGetenv(ctx context.Context, k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Criticalf(ctx, "%s environment variable not set.", k)
	}
	return v
}

func verifyApiToken(token string) error {
	for _, x := range apiTokens {
		if x == token {
			return nil
		}
	}
	return errors.New("invalid api token.")
}

type Input struct {
	ApiToken string `json:"api_token"`
	Logs     []struct {
		Type       string      `json:"type"`
		Attributes interface{} `json:"attributes"`
	} `json:"logs"`
}

type Output struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	output := Output{false, "something wrong."}
	code := 500

	defer func() {
		outjson, e := json.Marshal(output)
		if e != nil {
			log.Errorf(ctx, e.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		if code == 200 {
			fmt.Fprint(w, string(outjson))
		} else {
			http.Error(w, string(outjson), code)
		}
	}()

	if r.Method != "POST" {
		output.Message = "only POST method method was accepted"
		code = 404
		return
	}

	body, e := ioutil.ReadAll(r.Body)
	if e != nil {
		output.Message = e.Error()
		return
	}

	if len(body) > 64*1024 {
		output.Message = "Request payload size exceeded 64KB."
		code = 400
		return
	}
	input := Input{}
	e = json.Unmarshal(body, &input)
	if e != nil {
		output.Message = e.Error()
		log.Errorf(ctx, e.Error())
		code = 400
		return
	}

	if apiTokens == nil {
		apiTokens = strings.Split(mustGetenv(ctx, "API_TOKEN"), ",")
	}
	e = verifyApiToken(input.ApiToken)
	if e != nil {
		output.Message = e.Error()
		code = 401
		return
	}

	hc, err := google.DefaultClient(ctx, pubsub.PubsubScope)
	if err != nil {
		log.Criticalf(ctx, err.Error())
	}
	pubsubService, err := pubsub.New(hc)
	if err != nil {
		log.Criticalf(ctx, err.Error())
	}
	topicId := "projects/" + mustGetenv(ctx, "GCLOUD_PROJECT") + "/topics/" + mustGetenv(ctx, "PUBSUB_TOPIC")

	for _, entry := range input.Logs {
		json, e := json.Marshal(entry)
		if e != nil {
			output.Message = e.Error()
			log.Errorf(ctx, e.Error())
			code = 400
			return
		}
		msg := &pubsub.PublishRequest{
			Messages: []*pubsub.PubsubMessage{
				{
					Data: base64.StdEncoding.EncodeToString(json),
				},
			},
		}
		log.Infof(ctx, "publish data = %v", string(json))
		if _, err := pubsubService.Projects.Topics.Publish(topicId, msg).Do(); err != nil {
			output.Message = fmt.Sprintf("Could not publish message: %v", err)
			log.Errorf(ctx, "Could not publish message: %v", err)
			code = 500
			return
		}
	}

	output.Success = true
	output.Message = "ok"
	code = 200
	return
}
