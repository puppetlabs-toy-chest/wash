# Wash Website

## Development

### Setup

```
bundle install --path vendor/bundle
```

### Building the site

To start a local development server
```
bundle exec jekyll server --baseurl /wash
```

Go to `http://localhost:4000/wash/` to see the site running. Changes will be picked up automatically without restarting the server.

To just build the site:
```
bundle exec jekyll build
```

## Regenerate swagger docs

[api docs](./content/api/api.json) are generated using [go-swagger](https://github.com/go-swagger/go-swagger). Install `swagger` with
```
go get -u github.com/go-swagger/go-swagger/cmd/swagger
```

Then run it and [redoc-cli](https://github.com/Rebilly/ReDoc/blob/master/cli/README.md) from the project root to update the static docs page.
```
swagger generate spec > docs/docs/api.json
npx redoc-cli bundle docs/docs/api.json -o docs/docs/api.html --options.nativeScrollbars
```

## Extending the screencasts

You'll need to install [asciinema](https://asciinema.org/docs/installation) and [doitlive](https://github.com/sloria/doitlive).

All screencast scripts are located in `_screencasts/<category>/<script>`. To record a screencast, start-up a Wash shell then in that shell run

```
/path/to/_screencasts/record.sh <script>
```

`_screencasts/record.sh` will save the recorded screencast to `assets/<category>/<script_name>.cast`.
