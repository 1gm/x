# text-to-speech

Watches a directory for input text (specified by `-i`, defaults to "input") and processes it using
[AWS Polly](https://aws.amazon.com/polly/).

Will need to have AWS credentials on the machine, default setup is to just use the default profile.

See [configuring the Go SDK](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) for how to
customize the configuration.