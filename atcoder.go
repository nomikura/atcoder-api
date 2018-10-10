// package main

package hello

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// AtCoderのコンテスト情報を管理するモジュール
type AtCoder struct {
	Contests        []AtCoderContest
	RawContests     []RawAtCoderContest
	ContestDocument *goquery.Document
}

type RawAtCoderContest struct {
	Title     string
	Path      string
	StartTime string
	Duration  string
	Rated     string
}

// AtCoderのコンテスト情報
type AtCoderContest struct {
	Title     string
	Path      string
	StartTime int64
	Duration  int64
	Rated     string
}

// 流れ
func (atcoder *AtCoder) SetData() {
	// goqueryが使える状態にする
	ok := atcoder.SetContestDocument()
	if !ok {
		return
	}

	// 生コンテストデータ取得
	ok = atcoder.SetRawContest()
	if !ok {
		return
	}

	// コンテストデータ取得
	ok = atcoder.SetContest()
	if !ok {
		return
	}
}

// goqueryが使える状態にする
func (atcoder *AtCoder) SetContestDocument() bool {
	// GETリクエストを送る
	response, err := http.Get("https://beta.atcoder.jp/contests/?lang=ja")
	if err != nil {
		log.Printf("failed GET request: %v", err)
		return false
	}
	defer response.Body.Close()

	// HTMLを読み込む
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		fmt.Print(err)
		return false
	}

	atcoder.ContestDocument = doc
	return true
}

func (atcoder *AtCoder) SetRawContest() bool {
	var tableSelection *goquery.Selection
	atcoder.ContestDocument.Find("h3").Each(func(i int, s *goquery.Selection) {
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

	atcoder.RawContests = rawContest

	return true
}

func (atcoder *AtCoder) SetContest() bool {
	var contests []AtCoderContest
	for _, rawContest := range atcoder.RawContests {
		contests = append(contests, atcoder.Parse(rawContest))
	}
	atcoder.Contests = contests
	return true
}

func (atcoder *AtCoder) Parse(rawContest RawAtCoderContest) AtCoderContest {
	// Durationを求める
	str := strings.Replace(rawContest.Duration, ":", "h", 1) + "m"
	tim, _ := time.ParseDuration(str)
	duration := int64(tim.Seconds())

	// StartTimeを求める
	start := rawContest.StartTime
	atoi := func(str string) int {
		ret, _ := strconv.Atoi(str)
		return ret
	}
	// [2018-09-22 21:00:00+0900]の形式で抜き出した時間を無理矢理Timeオブジェクトにする
	year, month, day, hour, minute := atoi(start[:4]), atoi(start[5:7]), atoi(start[8:10]), atoi(start[11:13]), atoi(start[14:])
	// 取得する時間はJSTなので、日本時間をTimeオブジェクトにするように処理する
	jst, _ := time.LoadLocation("Asia/Tokyo")
	startTime := time.Date(year, time.Month(month), day, hour, minute, 0, 0, jst)
	unix := startTime.Unix()

	return AtCoderContest{
		Title:     rawContest.Title,
		Path:      rawContest.Path,
		StartTime: unix,
		Duration:  duration,
		Rated:     rawContest.Rated,
	}
}

func (atcoder *AtCoder) WriteContestData() {
	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(atcoder.Contests)
	if err != nil {
		log.Print(err)
	}
	err = ioutil.WriteFile("contest", buffer.Bytes(), 0600)
	if err != nil {
		log.Print(err)
	}
}

func (atcoder *AtCoder) ReadContestData() {
	raw, err := ioutil.ReadFile("contest")
	if err != nil {
		log.Print(err)
	}
	buffer := bytes.NewBuffer(raw)
	dec := gob.NewDecoder(buffer)
	err = dec.Decode(atcoder.Contests)
	if err != nil {
		log.Print()
	}
}

func sum() []AtCoderContest {
	var nomi []AtCoderContest
	nomi = append(nomi, AtCoderContest{Title: "hoge"})

	// GETリクエストを送る
	response, err := http.Get("https://beta.atcoder.jp/contests/?lang=ja")
	if err != nil {
		log.Printf("failed GET request: %v", err)
		return nomi
	}
	defer response.Body.Close()

	// HTMLを読み込む
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		fmt.Print(err)
	}

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

	return contests
}

func ParseSum(rawContest RawAtCoderContest) AtCoderContest {
	// Durationを求める
	str := strings.Replace(rawContest.Duration, ":", "h", 1) + "m"
	tim, _ := time.ParseDuration(str)
	duration := int64(tim.Seconds())

	// StartTimeを求める
	start := rawContest.StartTime
	atoi := func(str string) int {
		ret, _ := strconv.Atoi(str)
		return ret
	}
	// [2018-09-22 21:00:00+0900]の形式で抜き出した時間を無理矢理Timeオブジェクトにする
	year, month, day, hour, minute := atoi(start[:4]), atoi(start[5:7]), atoi(start[8:10]), atoi(start[11:13]), atoi(start[14:])
	// 取得する時間はJSTなので、日本時間をTimeオブジェクトにするように処理する
	jst, _ := time.LoadLocation("Asia/Tokyo")
	startTime := time.Date(year, time.Month(month), day, hour, minute, 0, 0, jst)
	unix := startTime.Unix()

	return AtCoderContest{
		Title:     rawContest.Title,
		Path:      rawContest.Path,
		StartTime: unix,
		Duration:  duration,
		Rated:     rawContest.Rated,
	}
}
