This is the source code for the `wash` website. It's built using hugo.

To regenerate the site from this directory:

```bash
$ rm -rf ../docs/*
$ hugo
$ tree ../docs
```

This will output the contents of the site into `../docs`, the directory we've
configured Github to serve our website from.

## Regenerate swagger docs

[api docs](./content/api/api.json) are generated using [go-swagger](https://github.com/go-swagger/go-swagger). Install `swagger` with
```
go get -u github.com/go-swagger/go-swagger/cmd/swagger
```

Then run it and [redoc-cli](https://github.com/Rebilly/ReDoc/blob/master/cli/README.md) from the project root to update the static docs page.
```
swagger generate spec > website/static/docs/api/api.json
npx redoc-cli bundle website/static/docs/api/api.json -o website/static/docs/api/index.html --options.nativeScrollbars
```

> Note that this is somewhat painful to get right with the current state of Go modules. Please ask for help if you have trouble.
