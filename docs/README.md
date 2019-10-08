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
