FROM golang:1.14 AS build

WORKDIR /workspace
COPY . /workspace/.
RUN go build -o bootstrap main.go

FROM vault:1.4.0
COPY --from=build /workspace/bootstrap bootstrap
ADD start.sh /start.sh
ENTRYPOINT ["/start.sh"]