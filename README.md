# godnsd (go-dns-discovery)

`godnsd` is a DNS server who fetch records dynamically form multiple sources, like filesystem or docker (labels).
It's easily to add other type of source.

## Use case

This service will discover DNS record from provider, and you can use this DNS server to resolve domain dynamically.

### Supported provider

#### Filesystem

Currently, the filesystem provider does not watch modification.

```yaml
# /etc/godnsd/config.yml
providers:
  exemple.local:
    type: fs
    config:
      path: "/app/exemple.local.yml"
  other.local:
    type: fs
    config:
      path: "/app/other.local.yml"
```

```yaml
# /app/exemple.local.yml
- name: 'foo.local'
  type: A
  value: 127.0.0.1

- name: '*.foo.local'
  type: CNAME
  value: foo.local.

- name: 'bar.local'
  type: CNAME
  value: foo.local.
```

#### Docker

The provider docker will watch docker events containers and refresh configuration.

```yaml
# /etc/godnsd/config.yml
providers:
  docker:
    type: docker
```

```yaml
# compose.yml
services:
  nginx:
    image: nginx:latest
    labels:
      - "godnsd.enable=true" # Mandatory

      # will use the internal IP of the 'default' network
      - "godnsd.records.test.name=foo.local" 
      - "godnsd.records.test.type=A"

      # will use the internal IP of the 'custom' network
      - "godnsd.records.db.name=bar.foo.local"
      - "godnsd.records.db.type=A"
      - "godnsd.records.db.network=custom"

      # will only declare a CNAME entry pointing to 'foo.local'
      - "godnsd.records.db.name=other.foo.local"
      - "godnsd.records.db.type=CNAME"
      - "godnsd.records.db.value=foo.local."
    networks:
      - default
      - custom
```

#### Api

The provider api will wait for http request to add record in memory (all record added will be lost when service is stopped).
To use it, you have to enable http server.

```yaml
# /etc/godnsd/config.yml
http:
  enable: true
  listen: 127.0.0.1:8080
  enable_provider: true
```

##### Endpoint

* `POST /api/records` -> Add a record
* `DELETE /api/records` -> Remove a record
* `POST /api/records/present` -> Add a record (Acme DNS challenge compatibility)
* `DELETE /api/records/cleanup` -> Remove a record (Acme DNS challenge compatibility)

All of these endpoints needs HTTP header `Content-Type`: `application/json`.

Body for /api/records [POST|DELETE]

```json
{
  "name": "_acme.foo.bar.local.",
  "type": "TXT",
  "value": "token"
}
```

Body for /api/records/present & /api/records/cleanup [POST]

```json
{
  "fqdn": "_acme.foo.bar.local.",
  "value": "token"
}
```

### Global configuration

`godnsd` can be configured to set log level or change default template used for README.md image.
For configure `godnsd` add [config.yml](doc/examples/config.yml) at the root of dir.

## Usage

```help
DNS server who discover records from providers

Usage:
  godnsd [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  start       Start DNS server
  version     Show version info

Flags:
  -c, --config string   Define config path
  -h, --help            help for godnsd
  -l, --level string    Define log level (default "INFO")

```

## API

When server HTTP is enabled the endpoint `GET /api/records` will be availlable.
This endpoint return all DNS records currently registered.

## Development

* Generate mock:

  ```bash
  ./bin/mock.sh
  ```

* Build the project:

  ```bash
  go build .
  ```

* Run tests:

  ```bash
  go test ./...
  ```

## Production

* Run godnsd with docker:

  ```bash
  docker run -v '${PWD}/config.yml:/etc/godnsd/config.yml:ro' -v '/var/run/docker.sock:/var/run/docker.sock:ro' -p '53:53/udp' alexandreh2ag/godnsd:${VERSION}
  ```

* Install binary to custom location:

  ```bash
  # Latest
  curl -L "https://github.com/alexandreh2ag/godnsd/releases/latest/download/godnsd_$(uname -s)_$(uname -m)" -o ${DESTINATION}/godnsd
  # or specific version
  curl -L "https://github.com/alexandreh2ag/godnsd/releases/download/${VERSION}/godnsd_$(uname -s)_$(uname -m)" -o ${DESTINATION}/godnsd
  
  chmod +x ${DESTINATION}/godnsd
  ```

## Config file config.yml

See documentation example for godnsd configuration [here](doc/examples/config.yml).

## Bash/ZSH Shell Completion

### Bash

Add in your ~/.bashrc :
```
eval "$(godnsd --completion-script-bash)"
```

### ZSH

Add in your ~/.zshrc :
```
eval "$(godnsd --completion-script-zsh)"
```

## License

MIT License, see [LICENSE](LICENSE.md).
