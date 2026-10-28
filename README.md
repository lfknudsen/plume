# Plume

A file upload server with support for HTTPS and multiple-file upload.

The project includes a small client, though the server can be compiled
separately if you intend to use another frontend.

## Setup

The first time the programme is executed, it will generate a configuration file
with some default values intended for local use. If you need the server to listen
to requests from outside localhost or over HTTPS, you will need to make some
changes to the values.

### Dependencies
Go 1.27

### Compile with Make and Install

If you have `make` and my [install](https://gitlab.com/lfknudsen/install) programme,
you can use the included Makefile:

To compile and install the server (output name `plume` or `plume.exe`):

```
make server
```

To compile and install the client (output name `upload` or `upload.exe`):

```
make client
```

To compile and install both:

```
make
```

### Manual compile

To compile the server (`plume`):
```
go build -C cmd/plume -o ../..
```

To compile the client (`upload`);
```
go build -C cmd/plume -o ../..
```

## Server Usage

```
plume [OPTIONS]
    Start the server.

OPTIONS:
  -h, --help    Print this help message and exit.
  -s, --https   Connect via HTTPS.
  -c, --config  Set path to the config file.
```

## Client Usage
```
upload [OPTIONS] <filename(s)...>
    Upload file(s) to the server.

OPTIONS:
-h, --help    Print this help message and exit.
-s, --https   Connect via HTTPS.
```
