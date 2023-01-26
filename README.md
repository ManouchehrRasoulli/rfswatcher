# rfswatcher

Remote filesystem watcher. this service provide ability to watch remote paths on configured server and clone these
changes into local client.

### Components

- `Server`
  this type of configuration provides a watcher over given path, and accept connection from configured clients.
- `Client`
  provice a `tcp` connection into configured server, and over change notifications download given changes from server
  and create local changes.

### Issues

Following issues resists in developed service and need to fixed.

- Unsecure connection.
- Session management with clients.
- Improve file transfer size.
- List files on first connection into server.
- Ignore duplicated events in client.
