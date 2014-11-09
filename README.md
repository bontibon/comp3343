# comp3343

Quick and dirty [Protocol Buffer](https://developers.google.com/protocol-buffers/)
example for COMP 3343 (Data Communications & Computer Networks).

Clients can:

- Send messages to the server to be stored in an individual mailbox
- Query a list of message IDs in a mailbox
- Fetch messages by ID from a mailbox

## Building

- Clone repository or download and extract archive
- Open directory
- Install prerequisites
    - [protoc](https://github.com/google/protobuf/)
    - [goprotobuf](https://code.google.com/p/goprotobuf/)
    - [github.com/mattn/go-sqlite3](github.com/mattn/go-sqlite3)
    - [github.com/spf13/cobra](github.com/spf13/cobra)
- `make`
- `./server`

## License

MIT
