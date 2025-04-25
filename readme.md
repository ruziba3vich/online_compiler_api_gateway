# 🧠 Online Compiler API Gateway

This project is an **API Gateway** for a gRPC-based online code compiler service, built using **Go (Golang)** and **Gin**, with streaming support for real-time code execution results.

---

## Project Overview

The API Gateway acts as a bridge between the front-end client (e.g., a web interface) and the back-end gRPC compiler service (e.g., a Python code executor). It provides a **WebSocket endpoint** that allows users to stream execution results of submitted code in real-time.

---

## Features

- **WebSocket API** for live code execution
- **Streaming support** — output is sent as soon as it's generated
- **gRPC Client** communicates with the backend compiler service
- Graceful startup/shutdown via **Uber FX**
- Modular and clean codebase using layered architecture

---

## WebSocket Endpoint

### `GET /execute`

This WebSocket endpoint handles real-time communication. Clients can connect and send source code to be executed. The response (output or errors) will be streamed back as they are generated.

---

## Code Format

To execute code, the client must send it in the following format:

**Example:**

```JSON

{
    "language": "paython", // choose the programming language here
    "code": "print(\"Hello, my telegram channel is t.me/Soliyev_talks\")" // write the source code here in the specified language
}

---------------------------------------------------------------------------------------

{
    "input": "some input" // if your code requires some input, pass the input values in this way
}

```

## Technologies Used

- Go (Gin Framework)
- gRPC
- Bi-directional streaming
- Protocol Buffers
- WebSocket
- Uber FX (Dependency Injection)
- Custom Logger

---

## Run the Project

### 1. Set Environment Variables

```bash
export PYTHON_SERVICE=host:7771
export GATEWAY_PORT=7772

