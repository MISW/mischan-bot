ARG go_version=1.19

FROM golang:${go_version} as tools

ENV GO111MODULE=on
ENV CGO_ENABLED=0
RUN go install sigs.k8s.io/kustomize/kustomize/v4@v4.5.5

FROM golang:${go_version} as builder

COPY . /work
ENV GO111MODULE=on
ENV CGO_ENABLED=0
RUN cd /work && go build -o /mischan-bot

FROM gcr.io/distroless/static:debug

COPY --from=tools /go/bin/kustomize /bin
COPY --from=builder /mischan-bot /bin

ENTRYPOINT [ "/bin/mischan-bot" ]
