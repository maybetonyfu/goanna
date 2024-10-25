# Build stage
FROM golang:1.23-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o goanna ./server

# Final stage
FROM python:3.13-bookworm
WORKDIR /root
RUN pip install --upgrade pip
COPY parser/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY parser/ ./
COPY --chmod=0755  --from=builder /app/goanna .
COPY --chmod=0755  run.sh run.sh

CMD ./run.sh