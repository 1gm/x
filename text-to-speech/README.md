# text-to-speech

Watches a directory for input text (specified by `-i`, defaults to "input") and processes it using
[AWS Polly](https://aws.amazon.com/polly/).

Will need to have AWS credentials on the machine, default setup is to just use the default profile.

See [configuring the Go SDK](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) for how to
customize the configuration.

## mac requirements

Run `xcode-select --install` to get `xcrun` which is required by the audio player library.

## running

From this directory execute: `go run .` which will create a folder called 'input' (assuming `-i` not specified). You can
then drop text files into the input directory and wait until the speaker plays the text (check the terminal output) if
you've been waiting a while.

You can simply echo text files into the directory, e.g.: `echo "hello world" > hello.txt`.