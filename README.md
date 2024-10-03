# valise

This is a collection of useful utilities for creating a production-ready golang backend
service.

## packages

### attr

Wrapper that tries to unify between `slog.Attr/slog.Value` and opentelemetry `attribute.KeyValue/attribute.Value`.

### env

Utility that allows for storing and loading environment variables using an AES key
and a JSON file.

### jsonschema

Utility that converts `reflect.Type` to a JSON schema.

### log

Wrapper around `slog` and `charmbracelet/log`

### rpc

A rpc framework built on top of `labstack/echo` and JSON schema.

### utils

A collection of very simple utilities.

### vctx

A context wrapper that provides a lot of utility functions.
