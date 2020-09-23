These are benchmarks to determine the best messaging system to use for the connected on-prem request proxy approach.

The benchmarks consist of three main parts:
- The browser page (`browser/index.html`). Opening this page runs a script that starts making websockets requests to the server. It also logs the time that it took to get a response from the server.
- The server (`(kafka|stan)/server/server.go`). The server hosts a websockets endpoint at ":7777/ws". When a message is sent to the server, it forwarded over the message bus on the "query-topic" channel. The server then waits for a response message to be published on the "reply-topic" channel. Finally, a response is sent back through the websocket. The amount of time this process takes is recorded and written to a text file when the server is closed by Ctrl-C.
- The connector (`(kafka|stan)/connector/satellite_connector.go`). This is a simple program that just waits for a message to be sent on the "query-topic" channel. When it receives the message, it just sends it back to the `reply-topic` channel.

To run:
1. Replace the appropriate Kafka/NATS addresses with your own addresses. This needs to be done in both the server and connector files.
2. Run both the server and connector. 
3. Update the serverAddr in `browser/script.js` to point to your server address. 
4. Open index.html. This should make 1000 websocket requests to your server. 
5. Once the server is done handling the requests, it will log the statistics. If the server is exited by Ctrl-C, it will also write the latencies to a text file to the directory called `(kafka|stan).txt`.  
