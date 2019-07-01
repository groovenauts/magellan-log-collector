# magellan-log-collector

## Run local server

```
goapp serve
```

## Deploy

Specify GCP project id and api tokens (comma separated).

```
gcloud --project=YOUR-PROJECT-ID app deploy app.yaml
```

## Upload package to deploy via Google App Engine Admin API

Run the following command To gather source files and make manifest file. `v1` stands for the version of application.

```
./makepkg.sh v1
```

Upload source files and a manifest file to gcs.

```
gsutil cp -R pkg/v1 gs://your-gae-repository/magellan-log-collector/v1
```
