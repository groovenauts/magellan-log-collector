# magellan-log-collector

## Run local server

```
export GO111MODULE=on
go run magellan-log-collector.go
```

## Deploy

Specify GCP project id and api tokens (comma separated).

```
export GO111MODULE=on
gcloud --project=YOUR-PROJECT-ID app deploy app.yaml
```

## Upload package to deploy via Google App Engine Admin API

Run the following command to gather source files and make manifest file, upload them to gcs.
`v1` stands for the version of application.

```
export GO111MODULE=on
./stage.sh your-gae-repository v1
```

This workflow is automated by [Release workflow](.github/workflows/release.yml),
triggered by tag push.

