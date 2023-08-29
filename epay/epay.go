package epay

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

var epay_url string = ""
var epay_pid string = ""
var epay_key string = ""
var epay_notify_url string = ""
var epay_return_url string = ""

// 生成订单号
func generateOrderNumber() string {
	// 获取当前时间
	now := time.Now()
	nowTime := now.Format("20060102150405") // 格式化为年月日时分秒

	// 生成随机数
	randomNum := rand.Intn(101) // 生成0到100之间的随机数

	// 格式化为两位数
	if randomNum <= 10 {
		randomNumStr := fmt.Sprintf("%02d", randomNum)
		return nowTime + randomNumStr
	}

	return nowTime + strconv.Itoa(randomNum)
}

func Submit(money int, pay_type string, chatid int64, db *sql.DB) (string, string) {
	out_trade_no := generateOrderNumber()
	epay_name := out_trade_no

	md5_raw := "money=" + strconv.Itoa(money) + "&name=" + epay_name + "&notify_url=" + epay_notify_url + "&out_trade_no=" + out_trade_no + "&pid=" + epay_pid + "&return_url=" + epay_return_url + "&type=" + pay_type + epay_key
	hash_1 := md5.Sum([]byte(md5_raw))
	md5 := hex.EncodeToString(hash_1[:])

	url_param := "money=" + strconv.Itoa(money) + "&name=" + epay_name + "&notify_url=" + epay_notify_url + "&out_trade_no=" + out_trade_no + "&pid=" + epay_pid + "&return_url=" + epay_return_url + "&type=" + pay_type + "&sign=" + md5 + "&sign_type=MD5"
	url_raw := epay_url + "?" + url_param

	return url_raw, out_trade_no
}
