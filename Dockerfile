FROM golang:1.13.4

LABEL repository="https://github.com/matoous/golangci-lint-action"
LABEL homepage="https://github.com/matoous/golangci-lint-action"
LABEL maintainer="Matouš Dzivjak <matousdzivjak@gmail.com>"

LABEL com.github.actions.name="Action - GolangCI Lint"
LABEL com.github.actions.description="Lint your Go code with GolangCI Lint"
LABEL com.github.actions.icon="code"
LABEL com.github.actions.color="blue"

ENV GOPROXY https://proxy.golang.org

RUN go get -v github.com/golangci/golangci-lint/cmd/golangci-lint
RUN go get -v github.com/matoous/golangci-lint-action

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
