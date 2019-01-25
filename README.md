# AtCoder API(informal)
- [AtCoder](https://atcoder.jp/)の全コンテスト情報を取得できます

## [Contest](https://atcoder-api.appspot.com/contests)
| Field            | Description                                                  |
| ---------------- | ------------------------------------------------------------ |
| id               | String.Contest id. Contest URL is https://beta.atcoder.jp/contests/{id} |
| title            | String. Contest title.                                       |
| startTimeSeconds | Integer.Contest start time in unix format.                   |
| durationSeconds  | Integer.Duration of the contest in seconds.                  |
| ratedRange       | String. Contest rated range.                                 |
