const serverAddr = "ws://<REPLACE_WITH_YOUR_SERVER_ADDRESS>:7777/ws";
const w = new WebSocket(serverAddr);

let startTime = 0
let i = 0;

w.onmessage = function() {
  diff = new Date() - startTime;
  console.log(diff + ' ms'); // Log the total round-trip time, including network latency.
  i++;
  if (i < 1000) {
    startTime = new Date()
    w.send('hello')
  }
}

w.onopen = function () {
  // Send an initial message to the websocket to check it is working.
  startTime = new Date()
  w.send('hello')
}
