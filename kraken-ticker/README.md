# kraken-ticker

Fetches data from [Kraken's public ticker API](https://docs.kraken.com/rest/#tag/Market-Data/operation/getTickerInformation) 
and writes it to a file on a minutely basis.

It will output the current & 24 hour BTC/USD vwap.

### use

```
# -o = output directory, defaults to data
# -mkdir = make the directory -o if it doesn't already exist
# -gz = gzip compress result files
go run main.go
```

There are 709 pairs returned from the API.

To find pairs you care about you can run the tool once and then do something like `cat result.json | jq -r '.result | keys[]' > pairs.txt`
