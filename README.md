# HttpWriter - Log sender for remote JSON/HTTP endpoints

## About

`httpwriter` package allows you to send JSON formatted log messages to log receivers such as [fluentbit](https://github.com/fluent/fluent-bit). It provides a few configuration options to tweak the behaviour. It uses a HTTP conneciton pool to send batches of log lines over to the remote endpoint. Keep Alives are enabled on the connections.

## Features

* Integrates well with `context.Context` and is cancellable.
* Tweakable configuration options
* Callback for error notifications

## Configuration Options

`HttpWriterOptions` contains configuration options for `httpwriter`.

| Option       | Description                                                                                  |
| ------------ | -------------------------------------------------------------------------------------------- |
| `BatchSize`  | Controls the number of JSON lines that will be sent with each HTTP call.                     |
| `BufferSize` | Controls the number of log lines that can be written without getting blocked.                |
| `ErrorFunc`  | Callback function to receive any errors encountered while sending the logs over the network. |

## Example Code

```golang
// Initialize
ctx, cancel := context.WithCancel(context.Background())
options := httpwriter.HttpWriterOptions{
    BufferCapacity: 25,
    BatchSize:      50,
}
w := httpwriter.New(s.ctx, s.mockServer.URL, &options)
jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})
logger := slog.New(jsonHandler)
logger.Info("Message", slog.Int("counter", 1))

// To shutdown the sender
cancel()
```
