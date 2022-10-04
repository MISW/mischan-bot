ARG go_version=1.19

FROM golang:${go_version} as tools

ENV GO111MODULE=on
ENV CGO_ENABLED=0
RUN go install sigs.k8s.io/kustomize/kustomize/v4@v4.5.5

COPY . /work
WORKDIR /work
ENV GO111MODULE=on
ENV CGO_ENABLED=0

CMD go run main.go
