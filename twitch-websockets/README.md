# twitch-websockets

A sample program connecting to Twitch's websocket pubsub API and which responds to channel point redemptions.

In order to run this you'll need an access token and a channel ID configured in a config file, see the 
`config.example.json` file for the expected format. You can specify the path to the config file to use by using
the `-c` flag, e.g.: `go run . -c path/to/config.json`.

To find your channel ID you can [go to this link](https://streamscharts.com/tools/convert-username) which accepts your 
username and returns your channel ID.

If accessToken is blank then you'll need to fill out the required credentials by registering your own application at the
[twitch developer console](https://dev.twitch.tv/console/apps).