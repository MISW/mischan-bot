ARG go_version=1.26

# development
FROM golang:${go_version} AS development

ARG kustomize_version=v5.1.1
RUN go install sigs.k8s.io/kustomize/kustomize/v5@${kustomize_version}

COPY . /mischan-bot

WORKDIR /mischan-bot

CMD go mod download \
  && CGO_ENABLED=0 go run main.go

# workspace
FROM golang:${go_version} AS workspace

ARG kustomize_version=v5.1.1
RUN go install sigs.k8s.io/kustomize/kustomize/v5@${kustomize_version}

COPY . /mischan-bot

WORKDIR /mischan-bot

RUN go mod download \
  && CGO_ENABLED=0 go build -buildmode pie -o /mischan-bot/mischan-bot

# production
FROM gcr.io/distroless/base:debug AS production

RUN ["/busybox/sh", "-c", "ln -s /busybox/sh /bin/sh"]
RUN ["/busybox/sh", "-c", "ln -s /bin/env /usr/bin/env"]

COPY --from=workspace /mischan-bot/mischan-bot /bin/mischan-bot
COPY --from=workspace /go/bin/kustomize /bin/kustomize

ENTRYPOINT ["/bin/mischan-bot"]
