// package main

package hello

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

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
	atcoder.LoadGob("contests")
	context.JSON(http.StatusOK, atcoder.Contests)
}

func Update(context *gin.Context) {
	var r *http.Request = context.Request
	ctx := appengine.NewContext(r)

	log.Infof(ctx, "GET!! (nomikura)") // アクセスログ

	atcoder := &AtCoder{}
	atcoder.GetAllContest(context)
	// responseContests = atcoder.Contests

}
