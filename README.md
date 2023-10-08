# enlightenkitsap

The site for https://enlightenkitsap.org

## development

Run the site using [Docker](https://docs.docker.com/engine/install/).
This tool runs the site in a container, automatically downloading the site and compiling it.

The site runs on a [Go](https://go.dev/) server.
It is comprised of two executable programs:
1. site.go, to compile the web pages
1. main.go, to serve the web pages

Build and run the site with `docker compose up --build`
