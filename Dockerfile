FROM golang:1.14.0

LABEL repository="https://github.com/matoous/golangci-lint-action"
LABEL homepage="https://github.com/matoous/golangci-lint-action"
LABEL maintainer="Matouš Dzivjak <matousdzivjak@gmail.com>"

LABEL com.github.actions.name="Action - GolangCI Lint"
LABEL com.github.actions.description="Lint your Go code with GolangCI Lint"
LABEL com.github.actions.icon="code"
LABEL com.github.actions.color="blue"

ARG golangci_lint_version=1.21.0

ENV GOPROXY https://proxy.golang.org

# NOTE: GolangCI-Lint README says "Please, do not install golangci-lint by go get"
# See: https://github.com/golangci/golangci-lint#go
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v${golangci_lint_version}

# Install from this repository
ADD . /source
WORKDIR /source
RUN go install .

WORKDIR /
COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
