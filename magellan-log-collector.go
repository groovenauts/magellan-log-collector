package main

import (
  "encoding/json"
  "errors"
  "fmt"
  "io/ioutil"
  "net/http"
  "os"
  "strings"
  "google.golang.org/appengine"
  "google.golang.org/appengine/log"
  "google.golang.org/cloud/pubsub"
  "golang.org/x/net/context"
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
  Logs []struct {
    Type string `json:"type"`
    Attributes interface{} `json:"attributes"`
  } `json:"logs"`
}

type Output struct {
  Success bool `json:"success"`
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

  client, err := pubsub.NewClient(ctx, mustGetenv(ctx, "GCLOUD_PROJECT"))
  if err != nil {
    log.Criticalf(ctx, err.Error())
  }
  topic := client.Topic(mustGetenv(ctx, "PUBSUB_TOPIC"))

  for _, entry := range input.Logs {
    json, e := json.Marshal(entry)
    if e != nil {
      output.Message = e.Error()
      log.Errorf(ctx, e.Error())
      code = 400
      return
    }
    msg := &pubsub.Message{
      Data: []byte(json),
    }
    log.Infof(ctx, "publish data = %v", string(msg.Data))
    if _, err := topic.Publish(ctx, msg); err != nil {
      output.Message = fmt.Sprintf("Could not publish message: %v", err)
      code = 500
      return
    }
  }

  output.Success = true
  output.Message = "ok"
  code = 200
  return
}
