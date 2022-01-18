FROM golang:1.17 as tools

ENV GO111MODULE=on
RUN go get sigs.k8s.io/kustomize/kustomize/v4@v4.4.1

COPY . /work
WORKDIR /work
ENV GO111MODULE=on
ENV CGO_ENABLED=0

CMD go run main.go