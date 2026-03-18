# Fyers Go Trading System - Commands

This file contains the common commands needed to test, build, and run the trading system.

## Running Tests

To run the unit tests for the Market Data Handler (RingBuffer, TickPool, TickStore):

```bash
# Run all tests in the project with verbose output
go test ./tests/... -v
```

## Running the Application

To run the main application (ensure `FYERS_APP_ID`, `FYERS_APP_SECRET`, and `FYERS_ACCESS_TOKEN` are set in your `.env` file first):

```bash
# Run the application directly
go run .
```

## Running Analytics

After running the application and capturing data in `ticks.txt`, you can analyze the latency and tick intervals by running the analytics module.

```bash
# Run the analytics report
go run ./analytics
```

## Dependency Management

To ensure all required Go modules are downloaded and the `go.mod` file is tidy:

```bash
# Clean up and download dependencies
go mod tidy
```

## Building

To compile the application into a binary executable:

```bash
# Build the executable
go build -o fyers-trading .
```
