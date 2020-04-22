Backed for Webix Document Manager
==================

### How to build

```shell script
go build
```


### How to start

Normal start
```shell script
./wfs-ls -data path/to/file/storage
```

Readonly mode

```shell script
./wfs-ls -readonly -data path/to/file/storage
```

Set upload limit

```shell script
./wfs-ls -upload 50000000 -data path/to/file/storage
```

Use external preview generator

```shell script
./wfs-ls -preview http://localhost:3201 -data path/to/file/storage
```

### Other ways of configuration

- config.yml in the app's folder

```yaml
uploadlimit: 10000000
root: ./
port: 80
readonly: false
preview: ""
```

- env vars

```shell script
APP_ROOT=/files APP_UPLOADLIMIT=300000 wfs-ls
```

