FROM golang:1.14 as tools

ENV GO111MODULE=on
RUN go get sigs.k8s.io/kustomize/kustomize/v3@v3.3.0

COPY . /work
WORKDIR /work
ENV GO111MODULE=on
ENV CGO_ENABLED=0

CMD go run main.go