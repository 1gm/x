# html-speaker

An example showing how to play audio on an HTML speaker. The backend will send over websocket
some base64 encoded MP3 data to the frontend periodically. This data will be played automatically.

To build the assets you must have node installed. Run `npm install`.

Next you can run `make && go run cmd/server/*.go` and go to `http://localhost:8081.