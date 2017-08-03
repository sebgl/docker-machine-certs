# docker-machine-certs
Tool to generate docker-machine certs for client and server, as done by `docker-machine create`.

## Usage
```
docker-machine-certs \
    --out-dir <machine/storage/path> \
    --server-ip <158.69.229.111> \
    --server-dns <my.server.com> \
    --machine-name <my.machine> \
    --ssh-key-path <private/ssh/key>
```

## Result
The following tree of files will be created:

```
├── certs
│   ├── ca-key.pem
│   ├── ca.pem
│   ├── cert.pem
│   └── key.pem
└── machines
    └── my.machine
        ├── ca.pem
        ├── cert.pem
        ├── config.json
        ├── id_rsa
        ├── key.pem
        ├── server-key.pem
        └── server.pem
```

## Server files
The following files can be copied on the server, in /etc/docker/:
```
- ca.pem
- server-key.pem
- server.pem
```