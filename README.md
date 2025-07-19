# rfswatcher

Remote filesystem watcher. this service provide ability to watch remote paths on configured server and clone these
changes into local client.

### Components

- `Server`
  this type of configuration provides a watcher over given path, and accept connection from configured clients.
- `Client`
  provice a `tcp` connection into configured server, and over change notifications download given changes from server
  and create local changes.

### Command line options

|option|description|default
|----|----|----|
|-c,-config|specify configuration file for service|config.yml|

### Server configuration

here is the server configuration file example:
```yaml
type: server
address: localhost:9901
path: /path/to/the/file/or/directory/you/want/to/watch
server:
  # optional
  tls:
    key: /path/to/private/key
    cert: /path/to/certificate

  # optional
  pwfile: /path/to/password-file
```

#### User management

to add/remove user, you should first set the `pwfile` in server config and run these commands:
```bash
# add user
rfswatcher -create-user

# delete user
rfswatcher -delete-user
```

### Client configuration

here is the server configuration file example:
```yaml
type: client
address: <server-address>
path: /path/you/want/to/save/files
client:
  # optional
  username: username
  password: password

  # optional
  tls: true
```

### Issues

Following issues resists in developed service and need to fixed.

- ~~Unsecure connection.~~
- ~~Session management with clients.~~
- Improve file transfer size.
- List files on first connection into server.
- Ignore duplicated events in client.
