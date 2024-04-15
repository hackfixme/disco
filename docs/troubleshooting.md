# Troubleshooting

### `command not found` when running `disco`

If you installed Disco by extracting the binary from a pre-built package, make sure that you placed the binary in a directory on your `$PATH`. On Linux/macOS you can confirm this with `which disco`.

If you installed Disco via `go install`, make sure that you precisely follow the [Go installation instructions](https://go.dev/doc/install) for your platform.
Specifically, ensure that the `$GOPATH/bin` directory is part of your `$PATH`. For example, you might want to add this to your shell's initialization file: `export PATH=$(go env GOPATH)/bin:$PATH`. See [this article](https://go.dev/wiki/SettingGOPATH) for more information.
