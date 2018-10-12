// package main

package hello

import (
	"context"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

type demo struct {
	client     *storage.Client
	bucketName string
	bucket     *storage.BucketHandle

	w   io.Writer
	ctx context.Context
	// cleanUp is a list of filenames that need cleaning up at the end of the demo.
	cleanUp []string
	// failed indicates that one or more of the demo steps failed.
	failed bool
}

// 本番用
func init() {
	// Todo: AtCoderを最初に1度呼ぶ
	r := gin.New()
	r.GET("/update", Update)
	r.GET("/json", Json)
	http.Handle("/", r)
}

func Json(context *gin.Context) {
	// var r *http.Request = context.Request
	// ctx := appengine.NewContext(r)

	atcoder := &AtCoder{}
	atcoder.Context = context // 強制的に設定してる。よくない
	// atcoder.LoadGob("contests")
	// ファイルから読み込む
	atcoder.FileIO("read")
	context.JSON(http.StatusOK, atcoder.Contests)
}

func Update(context *gin.Context) {
	var r *http.Request = context.Request
	ctx := appengine.NewContext(r)

	log.Infof(ctx, "GET!! (nomikura)") // アクセスログ

	atcoder := &AtCoder{}
	atcoder.SetContestData(context)
	// responseContests = atcoder.Contests

}
