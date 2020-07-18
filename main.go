package main

import (
	"flag"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/robfig/cron/v3"
	"github.com/ysmood/kit"
)

var clockInConf = flag.String("clockin", "0 12 * * *", "cron 语法的定时签到")
var topic = flag.String("topic", "", "要自动定时置顶的主题的 url，深夜不会触发 (10:00 - 24:00)")
var interval = flag.Duration("interval", 15*time.Minute, "自动点击置顶主题的间隔")
var locChina, _ = time.LoadLocation("Asia/Shanghai")
var lock = &sync.Mutex{}

func main() {
	flag.Parse()

	if *topic != "" {
		go func() {
			for {
				h := hour()
				if 9 < h && h < 24 {
					stickyTopic()
				}
				time.Sleep(*interval)
			}
		}()
	}

	if *clockInConf != "" {
		clockIn()

		scheduler := cron.New()
		kit.E(scheduler.AddFunc(*clockInConf, func() {
			clockIn()
		}))
		scheduler.Run()
	}
}

func stickyTopic() {
	if !isLoggedIn() {
		login()
	}

	lock.Lock()
	defer lock.Unlock()

	browser := newBrowser(true).Timeout(30 * time.Second)
	defer browser.Close()

	page := browser.Page(*topic)
	page.Element(".box")

	wait := page.HandleDialog(true, "")
	go page.ElementMatches(".box .fr a", "置顶 10 分钟").Click()
	wait()

	page.WaitRequestIdle()()

	kit.Log("置顶了", *topic)
}

func clockIn() {
	if !isLoggedIn() {
		login()
	}

	lock.Lock()
	defer lock.Unlock()

	browser := newBrowser(true)
	defer browser.Close()

	page := browser.Timeout(time.Minute).Page("https://www.v2ex.com/")

	page.Element(`[href="/mission/daily"]`).Click()

	el := page.ElementMatches("input", "领取 X 铜币", "span", "每日登录奖励已领取")

	if el.Matches("input") {
		el.Click()
		page.ElementMatches(".message", "已成功领取每日登录奖励")
		kit.Log("签到成功")
	} else {
		kit.Log("已经签过到了")
	}
}

func isLoggedIn() bool {
	lock.Lock()
	defer lock.Unlock()

	browser := newBrowser(true)
	defer browser.Close()

	return browser.Page("https://www.v2ex.com/signin").WaitLoad().HasMatches("a", "登出|Sign Out")
}

func login() {
	lock.Lock()
	defer lock.Unlock()

	browser := newBrowser(false)
	defer browser.Close()

	browser.Page("https://www.v2ex.com/signin").ElementMatches("a", "登出|Sign Out")
}

func newBrowser(headless bool) *rod.Browser {
	url := launcher.New().Headless(headless).UserDataDir("tmp/user").Launch()
	return rod.New().ControlURL(url).Connect()
}

func hour() int {
	return time.Now().In(locChina).Hour()
}
