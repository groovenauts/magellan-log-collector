package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/pubsub/v1"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	apiTokens []string
)

func mustGetenv(ctx context.Context, k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Printf("%s environment variable not set.", k)
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
	ctx := r.Context()
	output := Output{false, "something wrong."}
	code := 500

	defer func() {
		outjson, e := json.Marshal(output)
		if e != nil {
			log.Printf(e.Error())
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
		log.Printf(e.Error())
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
		log.Printf(err.Error())
	}
	pubsubService, err := pubsub.New(hc)
	if err != nil {
		log.Printf(err.Error())
	}
	topicId := "projects/" + mustGetenv(ctx, "GCLOUD_PROJECT") + "/topics/" + mustGetenv(ctx, "PUBSUB_TOPIC")

	for _, entry := range input.Logs {
		json, e := json.Marshal(entry)
		if e != nil {
			output.Message = e.Error()
			log.Printf(e.Error())
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
		if _, err := pubsubService.Projects.Topics.Publish(topicId, msg).Do(); err != nil {
			output.Message = fmt.Sprintf("Could not publish message: %v", err)
			log.Printf("Could not publish message: %v", err)
			code = 500
			return
		}
	}

	output.Success = true
	output.Message = "ok"
	code = 200
	return
}

func main() {
	apiTokens = nil
	http.HandleFunc("/", postHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
