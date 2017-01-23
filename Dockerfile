# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/fajarpnugroho/pengine

# Install dependency
RUN go get github.com/gin-gonic/gin
RUN go get github.com/dgrijalva/jwt-go
RUN go get github.com/lib/pq
RUN go get github.com/NaySoftware/go-fcm
RUN go install github.com/fajarpnugroho/pengine

# Run the outyet command by default when the container starts.
ENTRYPOINT /go/bin/pengine

# Document that the service listens on port 8080.
EXPOSE 8080
