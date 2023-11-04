# enlightenkitsap

The site for https://enlightenkitsap.org

## development

Run the site using [Docker](https://docs.docker.com/engine/install/).
This tool runs the site in a container, automatically downloading the site and compiling it.

Build and run the site with `docker compose up --build`


The site runs on a [Go](https://go.dev/) server.
It is comprised of two executable programs:
1. site.go, to compile the web pages
1. main.go, to serve the web pages

Remember to re-generate the site when developing the site on on a computer with Go installed.
Run the site with `go generate && go run enlightenkitsap.org`

[VSCode](https://code.visualstudio.com/) is a useful integrated development environment.

Build the site as a single executable to the build folder with `go generate && go build -o build/enlightenkitsap enlightenkitsap.org`

### file sizes

Resources should not bee too large.  The site will fail to build if resources are too large.

#### images

Images should be less than 50 kilobytes.  Resize images using [Imagemagick](https://imagemagick.org).  Change the percentage to resize a file to different dimensions
```
convert small.jpg -resize 40% SOURCE.jpg
```

### pdf

PDFs should be less than 10 megabytes.  Shrink PDFs with the [Ghostscript](https://ghostscript.com) command below.  Improve the file size by changing the `-dPDFSETTINGS=/` argument to screen/ebook/printer/default.
```
gs -sDEVICE=pdfwrite -dCompatibilityLevel=1.4 -dPDFSETTINGS=/printer -dNOPAUSE -dBATCH -sOutputFile=small.pdf SOURCE.pdf
```