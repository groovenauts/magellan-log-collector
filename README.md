# magellan-log-collector

## Run local server

```
goapp serve
```

## Deploy

Specify GCP project id and api tokens (comma separated).

```
appcfg.py -A YOUR-PROJECT-ID -E GCLOUD_PROJECT:YOUR-PROJECT-ID -E API_TOKEN:XXXXXXXX update .
```
