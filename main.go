// package main

package hello

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/PuerkitoBio/goquery"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

var responseContests []AtCoderContest

func init() {
	r := gin.New()
	r.GET("/update", Update)
	r.GET("/json", func(c *gin.Context) {
		c.JSON(http.StatusOK, responseContests)
	})
	http.Handle("/", r)
}

func Update(c *gin.Context) {
	var w http.ResponseWriter = c.Writer
	var r *http.Request = c.Request

	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

	log.Infof(ctx, "GET!! -nomikura-") // アクセスログ

	// GETリクエストを送る
	response, err := client.Get("https://beta.atcoder.jp/contests/?lang=ja")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// HTMLを読み込む
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		fmt.Print(err)
	}

	// 予定されたコンテストのテーブルを取得
	var tableSelection *goquery.Selection
	doc.Find("h3").Each(func(i int, s *goquery.Selection) {
		if h3 := s.Text(); h3 == "予定されたコンテスト" {
			tableSelection = s.Next()
		}
	})

	// コンテストデータを取得
	var rawContest []RawAtCoderContest
	tableSelection.Find("div > table > tbody > tr").Each(func(i int, trSelection *goquery.Selection) {
		// とりあえず文字列でテーブル情報を取得
		var href string
		var rawData [4]string
		trSelection.Find("td").Each(func(i int, tdSelection *goquery.Selection) {
			rawData[i] = tdSelection.Text()
			if i == 1 {
				href, _ = tdSelection.Find("a").Attr("href")
			}
		})
		rawContest = append(rawContest, RawAtCoderContest{
			StartTime: rawData[0],
			Title:     rawData[1],
			Duration:  rawData[2],
			Rated:     rawData[3],
			Path:      href,
		})
	})

	var contests []AtCoderContest
	for _, rawContest := range rawContest {
		contests = append(contests, ParseSum(rawContest))
	}

	responseContests = contests
}
