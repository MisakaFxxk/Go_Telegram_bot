package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	datebese "telegrambot_vip/datebase"
	epay "telegrambot_vip/epay"
	keycommad "telegrambot_vip/key"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/parnurzeal/gorequest"
)

type DB struct {
	*sql.DB
}

var flag int = 1
var register_status int = 0
var userStates map[int64]bool
var db *sql.DB
var emby_url = [...]string{""}
var emby_apikey = [...]string{""}

// TG命令多并发-info
func tg_info(chatid int64, db *sql.DB, bot *tgbotapi.BotAPI) {
	active_check := check_actived(chatid, db)
	if active_check == 1 {
		// TG账号已激活
		rows, err := db.Prepare("SELECT emby_userid, emby_name, server_num, create_date, out_date, status FROM user WHERE chatid = ?")
		if err != nil {
			panic(err.Error())
		}

		var emby_userid, emby_name, server_num, create_date sql.NullString
		var out_date, embyuri, embyuri_cdn string
		var status int
		err = rows.QueryRow(chatid).Scan(&emby_userid, &emby_name, &server_num, &create_date, &out_date, &status)
		if err != nil {
			panic(err.Error())
		}
		if emby_userid.Valid {
			if server_num.String == "0" {
				embyuri = ""
				embyuri_cdn = ""
			} else if server_num.String == "1" {
				embyuri = ""
				embyuri_cdn = ""
			}

			if status == 1 {
				msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("ChatId：%d\n\n注册用户名：%v\n\n到期时间：%v\n\n服务器地址：\n - 主机名： %s \n - 端口： 8096\n\n防污染地址(不定期更换)：\n - 主机名： %s \n - 端口： 8096\n\n续费方式：发送/buy，获取发卡网链接后进入购买卡密。\n\n切勿对外泄露服务器地址和账号，切勿买卖合租账号，发现即删号！", chatid, emby_name.String, out_date, embyuri, embyuri_cdn))
				bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("ChatId：%d\n\n注册用户名：%v\n\n到期时间：%v\n\n续费方式：发送/buy，获取发卡网链接后进入购买卡密。", chatid, emby_name.String, out_date))
				bot.Send(msg)
			}
		} else {
			msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("ChatId：%d\n\n注册用户名：尚未注册\n\n到期时间：%v\n\n续费方式：发送/buy，获取发卡网链接后进入购买卡密。", chatid, out_date))
			bot.Send(msg)
		}

	} else {
		// TG账号未激活
		msg := tgbotapi.NewMessage(chatid, "请先使用 /key 来激活TG账号！")
		bot.Send(msg)
	}
}

// TG命令多并发-reset
func tg_reset(chatid int64, receive_msg string, db *sql.DB, bot *tgbotapi.BotAPI) {
	register_check := check_allready_registered(chatid, db)
	if register_check == 1 {
		// 已注册
		receive_msg_split := strings.Split(receive_msg, " ")
		if len(receive_msg_split) == 2 {
			new_pw := receive_msg_split[1]
			command_check := password_reset(chatid, new_pw, db)
			if command_check == 1 {
				msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("密码重置成功，新密码为：%s，请妥善保存！", new_pw))
				bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(chatid, "未知错误，请联系管理人员解决！")
				bot.Send(msg)
			}
		} else {
			msg := tgbotapi.NewMessage(chatid, "格式错误，请发送 /reset 新密码")
			bot.Send(msg)
		}

	} else {
		// 未注册
		msg := tgbotapi.NewMessage(chatid, "请先注册账号！")
		bot.Send(msg)
	}
}

// TG命令多并发-key
func tg_key(chatid int64, receive_msg string, db *sql.DB, bot *tgbotapi.BotAPI) {
	//join_group_check := check_joingroup(chatid)
	join_group_check := 1
	if join_group_check != 1 {
		msg := tgbotapi.NewMessage(chatid, "请先加入官方群再注册！\n加群后请留意群内验证回答，未通过验证的账号将会被封禁！\n@MisakaF_Emby_chat1")
		bot.Send(msg)
	} else if join_group_check == 1 {
		receive_msg_split := strings.Split(receive_msg, " ")
		if len(receive_msg_split) == 2 {
			receive_keys := receive_msg_split[1]
			length := len(receive_keys)
			key_nums := length / 20

			//获取总卡密时长
			var key_sum float64
			for i := 0; i < key_nums; i++ {
				key := receive_keys[i*20 : (i+1)*20]
				key_check, months := keycommad.Key_check(key, db)
				months_int, _ := strconv.ParseFloat(months, 64)
				if key_check == 2 {
					msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("%s：KEY不存在！", key))
					bot.Send(msg)
				} else if key_check == 3 {
					msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("%s：KEY已被使用！", key))
					bot.Send(msg)
				} else if key_check == 1 {
					key_sum += months_int
					key_delete := keycommad.Key_delete(key, chatid, db)
					if key_delete == 1 {
						msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("%s：核销成功！", key))
						bot.Send(msg)
					} else {
						msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("%s：未知错误，Error Code：1001", key))
						bot.Send(msg)
					}
				}
			}

			if key_sum != 0 {
				//将卡密时长追加至TG账号
				key_sum = key_sum * 31
				active_check := check_actived(chatid, db)
				if active_check == 0 {
					//首次激活
					stmt, err := db.Prepare("insert into user (chatid,out_date,status) values (?,DATE_ADD(NOW(), interval ? day),1)")
					if err != nil {
						log.Fatal(err)
					}

					_, err = stmt.Exec(chatid, int64(key_sum))
					if err != nil {
						log.Fatal(err)
					} else {
						msg := tgbotapi.NewMessage(chatid, "KEY已全部核销，若一切正常请发送 /create 用户名 来创建账号。")
						bot.Send(msg)
					}
				} else if active_check == 1 {
					//非首次，续费
					command_back := xufei_all(chatid, int64(key_sum), db)
					if command_back == 1 {
						msg := tgbotapi.NewMessage(chatid, "KEY已全部核销，若一切正常请发送 /info 查询最新到期时间。")
						bot.Send(msg)
					}
				}
			} else {
				msg := tgbotapi.NewMessage(chatid, "未检测到任何有效卡密")
				bot.Send(msg)
			}

		} else {
			msg := tgbotapi.NewMessage(chatid, "格式错误，请发送 /key 卡密")
			bot.Send(msg)
		}
	}
}

// TG命令多并发-create
func tg_create(chatid int64, receive_msg string, db *sql.DB, bot *tgbotapi.BotAPI) {
	receive_msg_2 := strings.Split(receive_msg, " ")
	length := len(receive_msg_2)
	if length == 3 {
		create_check, create_back := create_account(chatid, receive_msg_2[1], receive_msg_2[2], db)
		if create_check == 1 {
			msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("注册成功！\n\n用户名： %s\n密码： %s\n\n开始使用前请先详细阅读本服Wiki： https://wiki.misakaf.org/ \n\n发送/info获取所有线路，请使用客户端播放，避免使用浏览器播放。\n\n关注更新通知频道： @MisakaF_Emby", receive_msg_2[1], receive_msg_2[2]))
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("检测到异常返回：%s", create_back))
			bot.Send(msg)
		}

	} else {
		msg := tgbotapi.NewMessage(chatid, "格式错误，请发送 /create 用户名 密码 来注册")
		bot.Send(msg)
	}
}

// TG命令多并发-易支付
func tg_epay(chatid int64, months int64, pay_type string, db *sql.DB, bot *tgbotapi.BotAPI) {
	money := months * 13
	pay_url, out_trade_no := epay.Submit(int(money), pay_type, chatid, db)

	// 数据库写入
	stmt, err := db.Prepare("insert into tg_pay (chatid,trade_no,out_trade_no,status,type,money,time) values (?,?,?,0,?,?,NOW())")
	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(chatid, out_trade_no, out_trade_no, pay_type, money)
	if err != nil {
		log.Fatal(err)
	} else {
		msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("订单号：%s\n支付地址(请于5分钟内完成支付)：%s", out_trade_no, pay_url))
		bot.Send(msg)
	}

}

// 获取当前Chatid对应的服务器信息
func get_server_info(chatid int64, db *sql.DB) (string, string, int) {
	var embyuri string
	var embykey string
	var check int
	//错误恢复
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error:", r)
		}
	}()

	// 执行查询
	rows, err := db.Query(fmt.Sprintf("SELECT server_num FROM user WHERE chatid = %d", chatid))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	// 处理查询结果
	var server_num sql.NullString
	for rows.Next() {
		err = rows.Scan(&server_num)
		if err != nil {
			log.Fatal(err)
		}

		if err = rows.Err(); err != nil {
			log.Fatal(err)
		}
	}
	if server_num.Valid {
		servernum, _ := strconv.Atoi(server_num.String)
		embyuri = emby_url[servernum]
		embykey = emby_apikey[servernum]
		check = 1
	} else {
		check = 0
	}

	return embyuri, embykey, check
}

// 账号续费-总入口
func xufei_all(chatid int64, num int64, db *sql.DB) int64 {
	status := 0
	//错误恢复
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error:", r)
		}
	}()

	check_daoqi := check_expired(chatid, db)
	if check_daoqi == 1 {
		//账号未到期，续费
		check_xufei := xufei_weidaoqi(chatid, num, db)
		if check_xufei == 1 {
			status = 1
		} else if check_xufei == 0 {
			status = 0
		}
	} else if check_daoqi == 0 {
		//账号到期，续费，计费时间从今日起
		command_check := xufei_daoqi(chatid, num, db)
		if command_check == 1 {
			status = 1
		}
	}

	return int64(status)
}

// 未到期账号续费
func xufei_weidaoqi(chatid int64, num int64, db *sql.DB) int {
	check := 0
	add_day := num
	//错误恢复
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error:", r)
		}
	}()

	// 更新表中的数据
	stmt, err := db.Prepare("UPDATE user set out_date = DATE_ADD(out_date, interval ? DAY) where chatid = ? ")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(add_day, chatid)
	if err != nil {
		log.Fatal(err)
	} else {
		check = 1
	}

	return check
}

// 到期账号续费
func xufei_daoqi(chatid int64, num int64, db *sql.DB) int {
	check := 0
	add_day := num
	embyuri, embykey, check := get_server_info(chatid, db)
	//错误恢复
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error:", r)
		}
	}()

	if check == 0 {
		// 没有注册账号，仅操作数据库
		stmt, err := db.Prepare("UPDATE user set out_date = DATE_ADD(now(), interval ? DAY) where chatid = ? ")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(add_day, chatid)
		if err != nil {
			log.Fatal(err)
		}

	} else if check == 1 {
		//查询Emby userid
		row := db.QueryRow(fmt.Sprintf("SELECT emby_userid FROM user WHERE chatid = %d", chatid))

		var userid string
		err := row.Scan(&userid)
		if err != nil {
			log.Fatal(err)
		}
		//数据库续费
		stmt, err := db.Prepare("UPDATE user set out_date = DATE_ADD(now(), interval ? DAY) where chatid = ? ")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(add_day, chatid)
		if err != nil {
			log.Fatal(err)
		} else {
			//Emby内激活账号
			res := gorequest.New()
			var data2 = `{"IsAdministrator":false,"IsHidden":true,"IsHiddenRemotely":true,"IsDisabled":false,"EnableRemoteControlOfOtherUsers":false,"EnableSharedDeviceControl":false,"EnableRemoteAccess":true,"EnableLiveTvManagement":false,"EnableLiveTvAccess":true,"EnableMediaPlayback":true,"EnableAudioPlaybackTranscoding":false,"EnableVideoPlaybackTranscoding":false,"EnablePlaybackRemuxing":false,"EnableContentDeletion":false,"EnableContentDownloading":false,"EnableSubtitleDownloading":false,"EnableSubtitleManagement":false,"EnableSyncTranscoding":false,"EnableMediaConversion":false,"EnableAllDevices":true,"SimultaneousStreamLimit":3}`
			resp, _, _ := res.Post(embyuri+"/emby/Users/"+userid+"/Policy?api_key="+embykey).Set("accept", "application/json").Set("Content-Type", "application/json").Send(data2).End()
			if resp.StatusCode == 204 {
				check = 1
			}
		}
	}

	return check
}

// 查询当前Chatid对应的账号是否到期
func check_expired(chatid int64, db *sql.DB) int {
	check := 0
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error:", r)
		}
	}()

	// 查询数据库中的数据
	rows, err := db.Prepare("SELECT out_date FROM user WHERE chatid = ?")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	// 执行查询语句
	var out_date string
	err = rows.QueryRow(chatid).Scan(&out_date)
	if err != nil {
		panic(err.Error())
	}

	// 将 date 对象转换为 time.Time 类型
	t, err := time.Parse("2006-01-02", out_date)
	if err != nil {
		log.Fatal(err)
	}

	// 获取当前日期并与 date 对象进行比较
	now := time.Now()
	if t.Before(now) {
		check = 0 //到期
	} else if t.After(now) {
		check = 1 //未到期
	}

	return check
}

// 检查是否已创建账号
func check_allready_registered(chatid int64, db *sql.DB) int {
	check := 0
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error:", r)
		}
	}()

	// 查询数据库中的数据
	rows, err := db.Prepare("SELECT emby_userid FROM user WHERE chatid = ?")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	// 执行查询语句
	var emby_userid sql.NullString
	err = rows.QueryRow(chatid).Scan(&emby_userid)
	if err != nil {
		panic(err.Error())
	}

	if emby_userid.Valid {
		check = 1 // 已注册
	} else {
		check = 0 // 未注册
	}
	return check
}

// 检查当前TG账号是否激活
func check_actived(chatid int64, db *sql.DB) int {
	check := 0
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error:", r)
		}
	}()

	// 准备查询语句
	stmtOut, err := db.Prepare("SELECT COUNT(*) FROM user WHERE chatid = ?")
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	// 执行查询语句
	var count int
	err = stmtOut.QueryRow(chatid).Scan(&count)
	if err != nil {
		panic(err.Error())
	}

	// 检查结果
	if count > 0 {
		check = 1 // 存在。账号已激活
	} else {
		check = 0 // 不存在，账号未激活
	}

	return check
}

// 检测当前TG号是否加群
func check_joingroup(chatid int64) int {
	check := 0
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error:", r)
		}
	}()

	bot, err := tgbotapi.NewBotAPI("")
	if err != nil {
		panic(err)
	}

	var groupID int64 = 123456 // 群组的 Telegram ID

	ChatConfigWithUser := tgbotapi.ChatConfigWithUser{
		ChatID: groupID,
		UserID: chatid,
	}

	getChatMemberConfig := tgbotapi.GetChatMemberConfig{ChatConfigWithUser: ChatConfigWithUser}

	chatMember, err := bot.GetChatMember(getChatMemberConfig)
	if err != nil {
		panic(err)
	}

	if chatMember.IsMember || chatMember.IsCreator() || chatMember.IsAdministrator() {
		check = 1
	} else {
		check = 0
	}

	return check
}

// 重置密码
func password_reset(chatid int64, newpw string, db *sql.DB) int {
	check := 0
	embyuri, embykey, status := get_server_info(chatid, db)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error:", r)
		}
	}()

	if status == 1 {
		// 准备查询语句
		stmtOut, err := db.Prepare("SELECT emby_userid FROM user WHERE chatid = ?")
		if err != nil {
			panic(err.Error())
		}
		defer stmtOut.Close()

		// 执行查询语句
		var emby_userid string
		err = stmtOut.QueryRow(chatid).Scan(&emby_userid)
		if err != nil {
			panic(err.Error())
		}

		res_pw := gorequest.New()
		var data_pw = `{"ResetPassword" : true}`
		respon_pw, _, _ := res_pw.Post(embyuri+"/emby/Users/"+emby_userid+"/Password?api_key="+embykey).Set("accept", "application/json").Set("Content-Type", "application/json").Send(data_pw).End()
		if respon_pw.StatusCode == 204 {
			res := gorequest.New()
			var data = `{"CurrentPw":"" , "NewPw":"` + newpw + `","ResetPassword" : false}`
			respon, _, _ := res.Post(embyuri+"/emby/Users/"+emby_userid+"/Password?api_key="+embykey).Set("accept", "application/json").Set("Content-Type", "application/json").Send(data).End()
			if respon.StatusCode == 204 {
				check = 1
			}
		}
	} else if status == 0 {
		check = 0
	}
	return check
}

// 创建账号
func create_account(chatid int64, name string, password string, db *sql.DB) (int, string) {
	check := 0
	up_string := "None"
	check_active := check_actived(chatid, db)
	check_expire := check_expired(chatid, db)
	check_register := check_allready_registered(chatid, db)
	var embyuri, embykey, flag_tmp string
	// flag = flag * -1
	// if flag == -1 {
	// 	embyuri = emby_url[0]
	// 	embykey = emby_apikey[0]
	// 	flag_tmp = "0"
	// } else if flag == 1 {
	// 	embyuri = emby_url[1]
	// 	embykey = emby_apikey[1]
	// 	flag_tmp = "1"
	// }

	embyuri = emby_url[0]
	embykey = emby_apikey[0]
	flag_tmp = "0"

	if check_active == 1 && check_expire == 1 && check_register == 0 {
		//开始注册
		req := gorequest.New()
		var data string = `{"Name":"` + name + `"}`
		resp, body, _ := req.Post(embyuri+"/emby/Users/New?api_key="+embykey).Set("accept", "application/json").Set("Content-Type", "application/json").Send(data).End()
		if resp.StatusCode == 200 {
			var resjson map[string]interface{}
			err := json.Unmarshal([]byte(body), &resjson)
			if err != nil {
				panic(err)
			}
			emby_userid := resjson["Id"].(string)
			//创建账号结束，开始设置配置账号
			res := gorequest.New()
			var data2 = `{"IsAdministrator":false,"IsHidden":true,"IsHiddenRemotely":true,"IsDisabled":false,"EnableRemoteControlOfOtherUsers":false,"EnableSharedDeviceControl":false,"EnableRemoteAccess":true,"EnableLiveTvManagement":false,"EnableLiveTvAccess":true,"EnableMediaPlayback":true,"EnableAudioPlaybackTranscoding":false,"EnableVideoPlaybackTranscoding":false,"EnablePlaybackRemuxing":false,"EnableContentDeletion":false,"EnableContentDownloading":false,"EnableSubtitleDownloading":false,"EnableSubtitleManagement":false,"EnableSyncTranscoding":false,"EnableMediaConversion":false,"EnableAllDevices":true,"SimultaneousStreamLimit":3}`
			resp, _, _ := res.Post(embyuri+"/emby/Users/"+emby_userid+"/Policy?api_key="+embykey).Set("accept", "application/json").Set("Content-Type", "application/json").Send(data2).End()
			if resp.StatusCode == 204 {
				//账号配置完毕，开始设置密码
				res_pw := gorequest.New()
				var data_pw = `{"CurrentPw":"" , "NewPw":"` + password + `","ResetPassword" : false}`
				respon_pw, _, _ := res_pw.Post(embyuri+"/emby/Users/"+emby_userid+"/Password?api_key="+embykey).Set("accept", "application/json").Set("Content-Type", "application/json").Send(data_pw).End()
				if respon_pw.StatusCode == 204 {
					//密码配置完毕，开始写入数据库
					stmt, err := db.Prepare("UPDATE user set emby_userid = ? , emby_name = ?,server_num = ?,create_date = DATE(NOW()) where chatid = ? ")
					if err != nil {
						log.Fatal(err)
					}
					defer stmt.Close()

					_, err = stmt.Exec(emby_userid, name, flag_tmp, chatid)
					if err != nil {
						up_string = "账号注册成功！但数据库写入失败，请联系 @MisakaF处理，否则账号将会被清除。"
						log.Fatal(err)
					} else {
						check = 1
						up_string = fmt.Sprintf("注册成功！\n\n用户名： %s\n密码： %s \n\n开始使用前请先详细阅读本服Wiki： https://wiki.misakaf.org/ \n\n发送 /info 获取所有线路，请使用客户端播放，避免使用浏览器播放。\n\n关注更新通知频道： @MisakaF_Emby", name, password)
					}
				}
			}
		} else {
			up_string = body
		}
	} else if check_active == 0 {
		up_string = "账号未激活，请先发送 /buy 购买卡密来激活账号"
	} else if check_expire == 0 {
		up_string = "账号已过期，无法注册，请先发送 /key 卡密 来续费账号"
	} else if check_register == 1 {
		up_string = "您已经注册过账号了！"
	}

	return check, up_string
}

func main() {
	//初始化数据库
	db, err := datebese.NewDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//初始化BOT
	bot, err := tgbotapi.NewBotAPI("")
	if err != nil {
		log.Panic(err)
	}

	//错误恢复
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from error:", r)
		}
	}()

	bot.Debug = false

	userStates = make(map[int64]bool)

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// 获取用户输入的命令
		command := update.Message.Command()
		chatID := update.Message.Chat.ID

		if command == "start" {
			log.Println(update.Message.Chat.ID, update.Message.Chat.UserName, "|", update.Message.Text)
			chatid := update.Message.Chat.ID
			msg := tgbotapi.NewMessage(chatid, "请输入：/create 用户名 密码 来注册\n\n例如通过发送：/create helloworld 123456 来注册一个用户名为helloworld密码为123456的账号(请勿包含中文与特殊符号！)")
			bot.Send(msg)
		}

		if command == "help" {
			log.Println(update.Message.Chat.ID, update.Message.Chat.UserName, "|", update.Message.Text)
			chatid := update.Message.Chat.ID
			msg := tgbotapi.NewMessage(chatid, fmt.Sprintf("chatid: %d \n\n创建账号：/create 用户名 密码\n例：/create username password\n\n重置密码：/reset 密码\n\n查询账号信息：/info\n\n切换线路：/line\n\n获取续费网址：/buy", chatid))
			bot.Send((msg))
		}

		if command == "info" {
			log.Println(update.Message.Chat.ID, update.Message.Chat.UserName, "|", update.Message.Text)
			chatid := update.Message.Chat.ID
			go tg_info(chatid, db, bot)
		}

		if command == "reset" {
			log.Println(update.Message.Chat.ID, update.Message.Chat.UserName, "|", update.Message.Text)
			chatid := update.Message.Chat.ID
			go tg_reset(chatid, update.Message.Text, db, bot)
		}

		if command == "key" {
			log.Println(update.Message.Chat.ID, update.Message.Chat.UserName, "|", update.Message.Text)
			chatid := update.Message.Chat.ID
			go tg_key(chatid, update.Message.Text, db, bot)
		}

		if command == "create" {
			log.Println(update.Message.Chat.ID, update.Message.Chat.UserName, "|", update.Message.Text)
			// msg := tgbotapi.NewMessage(chatID, "测试中，尚未开放注册，请耐心等待")
			// bot.Send((msg))
			chatid := update.Message.Chat.ID
			go tg_create(chatid, update.Message.Text, db, bot)
		}

		if update.Message.Text == "/buy" {
			check_active := check_actived(chatID, db)
			if check_active == 0 && register_status == 0 {
				msg := tgbotapi.NewMessage(chatID, "尚未开放注册，请耐心等待")
				bot.Send(msg)
			} else {
				userStates[chatID] = true
				msg := tgbotapi.NewMessage(chatID, "请选择购买套餐：")
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("月卡 - 支付宝"),
						tgbotapi.NewKeyboardButton("月卡 - 微信"),
						// tgbotapi.NewKeyboardButton("季卡 - 支付宝"),
						// tgbotapi.NewKeyboardButton("季卡 - 微信"),
					),
				)
				bot.Send(msg)
			}
		} else if userStates[chatID] {
			if update.Message.Text == "月卡 - 支付宝" {
				go tg_epay(chatID, 1, "alipay", db, bot)
				delete(userStates, chatID)
			} else if update.Message.Text == "月卡 - 微信" {
				go tg_epay(chatID, 1, "wxpay", db, bot)
				delete(userStates, chatID)
			} else if update.Message.Text == "季卡 - 支付宝" {
				go tg_epay(chatID, 3, "alipay", db, bot)
				delete(userStates, chatID)
			} else if update.Message.Text == "季卡 - 微信" {
				go tg_epay(chatID, 3, "wxpay", db, bot)
				delete(userStates, chatID)
			}
		}

	}
}
