// package main

package hello

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/appengine/file"

	"github.com/gin-gonic/gin"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"github.com/PuerkitoBio/goquery"
)

// AtCoderのコンテスト情報を管理するモジュール
type AtCoder struct {
	Contests    []AtCoderContest
	RawContests []RawAtCoderContest
	HttpClient  *http.Client
	Context     *gin.Context
}

type RawAtCoderContest struct {
	ID        string
	Title     string
	StartTime string
	Duration  string
	Rated     string
}

// AtCoderのコンテスト情報
type AtCoderContest struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	StartTime int64  `json:"startTimeSeconds"`
	Duration  int64  `json:"durationSeconds"`
	Rated     string `json:"ratedRange"`
}

func (atcoder *AtCoder) FileIO(operate string) {
	var r *http.Request = atcoder.Context.Request
	var w http.ResponseWriter = atcoder.Context.Writer
	ctx := appengine.NewContext(r)

	// デフォルトのバケットを指定する(App Engineのコンテストから取得できる)
	bucket, err := file.DefaultBucketName(ctx)
	if err != nil {
		log.Errorf(ctx, "Faild to get default GCS bucket name: %v", err)
	}

	// clientをつくる
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Errorf(ctx, "Faild to create client: %v", err)
	}
	defer client.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	buf := &bytes.Buffer{}
	d := &demo{
		w:          buf,
		ctx:        ctx,
		client:     client,
		bucket:     client.Bucket(bucket),
		bucketName: bucket,
	}

	fileName := "demo-testfile-go"

	if operate == "write" {
		// コンテストデータをバイナリにエンコード
		var binaryData []byte
		Encode(atcoder.Contests, &binaryData)
		// ファイルに書き込む
		d.createFile(fileName, binaryData)
		log.Infof(ctx, "Write to file")
	} else if operate == "read" {
		// ファイルをバイナリで読み込む
		var binaryData []byte
		d.readFile(fileName, &binaryData)
		// バイナリをコンテストデータにデコード
		Decode(binaryData, &atcoder.Contests)
		log.Infof(ctx, "Read file")
	} else {
		log.Infof(ctx, "operation is not read and write")
	}

	if d.failed {
		w.WriteHeader(http.StatusInternalServerError)
		buf.WriteTo(w)
	} else {
		w.WriteHeader(http.StatusOK)
		buf.WriteTo(w)
	}
}

//[START write]
func (d *demo) createFile(fileName string, byteDataToWrite []byte) {
	wc := d.bucket.Object(fileName).NewWriter(d.ctx)
	wc.ContentType = "text/plain"
	wc.Metadata = map[string]string{
		"x-goog-meta-foo": "foo",
		"x-goog-meta-bar": "bar",
	}
	d.cleanUp = append(d.cleanUp, fileName)

	// 書き込む
	if _, err := wc.Write(byteDataToWrite); err != nil {
		d.errorf("createFile: unable to write data to bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}

	// ファイル閉じてるのかな？これがないと書き込めない
	if err := wc.Close(); err != nil {
		d.errorf("createFile: unable to close bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
}

//[END write]

//[START read]
func (d *demo) readFile(fileName string, data *[]byte) {
	// ファイルを開く
	rc, err := d.bucket.Object(fileName).NewReader(d.ctx)
	if err != nil {
		d.errorf("readFile: unable to open file from bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
	defer rc.Close()

	// データを読み込む
	slurp, err := ioutil.ReadAll(rc)
	if err != nil {
		d.errorf("readFile: unable to read data from bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}

	*data = slurp
}

//[END read]

// dataをbyteArrayにエンコードする
func Encode(data interface{}, byteArray *[]byte) {
	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(data)
	if err != nil {
		// log.Print("encode: ", err)
	}

	*byteArray = buffer.Bytes()
}

// byteArrayをdataにデコードする(dataはポインタ型)
func Decode(byteArray []byte, data interface{}) {
	buffer := bytes.NewBuffer(byteArray)
	dec := gob.NewDecoder(buffer)
	err := dec.Decode(data)
	if err != nil {
		// log.Print("decode: ", err)
	}
}

func (d *demo) errorf(format string, args ...interface{}) {
	d.failed = true
	fmt.Fprintln(d.w, fmt.Sprintf(format, args...))
	log.Errorf(d.ctx, format, args...)
}

func (atcoder *AtCoder) SetContestData(context *gin.Context) {
	// atcoder.HttpClient = client
	atcoder.Context = context

	// 予定されたコンテストの生データを取得
	atcoder.GetFutureContest()

	// 過去のコンテストの生データを取得
	atcoder.GetPastContest()

	// 5. 生のコンテストデータをパースする
	for _, rawContest := range atcoder.RawContests {
		atcoder.Contests = append(atcoder.Contests, ParseSum(rawContest))
	}

	sort.Slice(atcoder.Contests, func(i, j int) bool { return atcoder.Contests[i].StartTime < atcoder.Contests[j].StartTime })

	// ファイルに書き込む
	atcoder.FileIO("write")
	// atcoder.StoreGob("contests")
}

// 予定されたコンテストデータを取得する
func (atcoder *AtCoder) GetFutureContest() {
	// var responseWriter http.ResponseWriter = atcoder.Context.Writer
	var request *http.Request = atcoder.Context.Request

	context := appengine.NewContext(request)
	client := urlfetch.Client(context)

	// 1. GETリクエスト
	response, err := client.Get("https://beta.atcoder.jp/contests/?lang=ja")
	time.Sleep(2 * time.Second)
	if err != nil {
		fmt.Print(err)
		return
	}

	// 2. goqueryを使えるようにする
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		fmt.Print(err)
	}

	// 3. 予定されたコンテストのテーブルを取得
	var tableSelection *goquery.Selection
	doc.Find("h3").Each(func(i int, s *goquery.Selection) {
		if h3 := s.Text(); h3 == "予定されたコンテスト" {
			tableSelection = s.Next()
		}
	})

	// 4. 生のコンテストデータを取得
	atcoder.GetRawContestFromTable(tableSelection)

	// 5. 生のコンテストデータをパースする
	// for _, rawContest := range atcoder.RawContests {
	// 	atcoder.Contests = append(atcoder.Contests, ParseSum(rawContest))
	// }

}

// 過去のコンテストデータを取得する
func (atcoder *AtCoder) GetPastContest() {
	baseURL := "https://beta.atcoder.jp/contests/archive?lang=ja"
	// 準備
	var request *http.Request = atcoder.Context.Request
	context := appengine.NewContext(request)
	client := urlfetch.Client(context)

	numberOfPage, ok := atcoder.GetNumberOfPage(baseURL)
	// ページ番号の取得に失敗
	if !ok {
		log.Infof(context, "Faild to get number of page!!")
		return
	}

	for page := 1; page <= numberOfPage; page++ {
		// 1. GETリクエスト
		response, err := client.Get(baseURL + "&page=" + strconv.Itoa(page))
		time.Sleep(2 * time.Second)
		if err != nil {
			fmt.Print(err)
			return
		}

		// 2. goqueryを使えるようにする
		doc, err := goquery.NewDocumentFromReader(response.Body)
		if err != nil {
			fmt.Print(err)
		}

		// 3. 予定されたコンテストのテーブルを取得
		var tableSelection *goquery.Selection = doc.Find("table")

		// 4. 生のコンテストデータを取得
		atcoder.GetRawContestFromTable(tableSelection)

		// 5. 生のコンテストデータをパースする
		// for _, rawContest := range atcoder.RawContests {
		// 	atcoder.Contests = append(atcoder.Contests, ParseSum(rawContest))
		// }
	}

}

func (atcoder *AtCoder) GetNumberOfPage(baseURL string) (int, bool) {
	var request *http.Request = atcoder.Context.Request

	context := appengine.NewContext(request)
	client := urlfetch.Client(context)

	// 1. Getリクエスト
	response, err := client.Get(baseURL)
	time.Sleep(2 * time.Second)
	if err != nil {
		fmt.Print(err)
		return 0, false
	}

	// 2. goqueryを使えるようにする
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		fmt.Print(err)
		return 0, false
	}

	// 3. 番号を取得
	numberOfPage := 1
	doc.Find("ul > li > a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		// aタグにhrefが存在しない
		if !exists {
			return
		}
		// hrefに[page=]が含まれない
		if !strings.Contains(href, "page=") {
			return
		}

		// タグの中身を数値にできる
		if page, err := strconv.Atoi(s.Text()); err == nil {
			// log.Infof(context, "pageNumger(nomikura): %+v", page)
			if page > numberOfPage {
				numberOfPage = page
			}
		}

	})
	// log.Infof(context, "pageSize(nomikura): %+v", pageSize)
	return numberOfPage, true
}

// 生のコンテストデータをパースする
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
		ID:        rawContest.ID,
		StartTime: unix,
		Duration:  duration,
		Rated:     rawContest.Rated,
	}
}

// 生のコンテストデータをセットする
func (atcoder *AtCoder) GetRawContestFromTable(tableSelection *goquery.Selection) {
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
		rawContest := RawAtCoderContest{
			StartTime: rawData[0],
			Title:     rawData[1],
			Duration:  rawData[2],
			Rated:     rawData[3],
			ID:        href[10:],
		}
		atcoder.RawContests = append(atcoder.RawContests, rawContest)
	})
}
