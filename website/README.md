This is the source code for the `wash` website. It's built using hugo.

To regenerate the site from this directory:

```bash
$ rm -rf ../docs/*
$ hugo
$ tree ../docs
```

This will output the contents of the site into `../docs`, the directory we've
configured Github to serve our website from.
