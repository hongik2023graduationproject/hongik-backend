# Stage 1: Build Go application
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 2: Build Hong-Ik interpreter (C++)
FROM alpine:3.19 AS hongik-builder

WORKDIR /build

RUN apk add --no-cache cmake build-base g++

COPY hong-ik/ ./
RUN cmake -B build -DCMAKE_BUILD_TYPE=Release && \
    cmake --build build

# Stage 3: Final runtime image
FROM alpine:3.19

WORKDIR /app

RUN apk --no-cache add ca-certificates libc6-compat libstdc++

COPY --from=builder /app/server .
COPY --from=hongik-builder /build/build/HongIk ./HongIk

RUN chmod +x ./server ./HongIk

ENV PORT=8080
ENV ENV=production
ENV INTERPRETER_PATH=/app/HongIk
ENV CORS_ORIGINS=http://localhost:3000,http://localhost:5173
ENV MAX_CONCURRENT_EXEC=5

EXPOSE 8080

CMD ["./server"]
