# to-sidecar-or-not

An exploration of the pros and cons of using the sidecar model for common cross-langauge functional needs.


## Contents

- [Project Layout](#project-layout)

## Project Layout

The project is entirely written in Go, and structured similarly to what's described in Ben Johnson's [Structuring Applications in Go](https://medium.com/@benbjohnson/structuring-applications-in-go-3b04be4ff091) article.

There are two programs currently built by code in this repo.

### Server

This is a simple HTTP server with a `/ping` endpoint.

[Main file](cmd/server/main.go)

### Sidecar

This is a simple reverse proxy that forwards requests on the `/ping` endpoint to something listening on another local port at the same path.

[Main file](cmd/sidecar/main.go)