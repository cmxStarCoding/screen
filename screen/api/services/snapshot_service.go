package services

import (
	"context"
	"crm.com/common/cache"
	"crm.com/common/database"
	utils2 "crm.com/common/utils"
	"crm.com/screen/api/model"
	"crm.com/screen/api/utils"
	"crm.com/screen/api/validator/snapshot"
	"crm.com/screen/library"
	"crm.com/screen/queue"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"gorm.io/gorm"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"time"
)

// SnapshotService 截图服务
type SnapshotService struct{}

func (s SnapshotService) DoTask(requestData *snapshot.DoTaskRequest) {

	jsonByte, _ := json.Marshal(requestData)
	//
	if requestData.MediaType == 2 {
		//字节充值、转款、退款截图队列
		zjRabbit := queue.NewRabbitMQ("yoyo_zj_exchange", "yoyo_zj1_route", "yoyo_zj1_queue")
		//defer zjRabbit.Close()
		//上线前这里需要改成5
		for i := 0; i < 5; i++ {
			if i == 0 {
				zjRabbit.SendMessage(queue.Message{Body: string(jsonByte)})
				zjRabbit.SendDelayMessage(queue.Message{Body: string(jsonByte), DelayTime: 30})
			} else {
				zjRabbit.SendDelayMessage(queue.Message{Body: string(jsonByte), DelayTime: (i * 60) + 30})
			}
		}
	}

	if requestData.MediaType == 1 {
		//快手充值、转款、退款截图队列
		ksRabbit := queue.NewRabbitMQ("yoyo_ks_exchange", "yoyo_ks_route", "yoyo_ks_queue")
		//defer ksRabbit.Close()
		ksRabbit.SendMessage(queue.Message{Body: string(jsonByte)})
		//ksRabbit.SendDelayMessage(queue.Message{Body:  string(jsonByte), DelayTime: 2})
	}
}

// SnapshotZjFlow 字节月流水截图
func (s SnapshotService) SnapshotZjFlow(CmsMediaAccount model.CmsMediaAccount) string {
	lastMonth := utils.GetLastMonth("200601")
	failMsg := fmt.Sprintf("字节月度流水截图错误:月份：%v，产品id：%v，媒体账号id：%v，错误原因：", lastMonth, CmsMediaAccount.ProductID, CmsMediaAccount.AdvertiserID)
	defer func() {
		if r := recover(); r != nil {
			log.Println(failMsg+"%v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	// 创建Chrome无头浏览器选项
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("window-size", "1920,1080"),
	)
	// 初始化浏览器
	ctx, cancel = chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()
	// 创建Chrome浏览器上下文
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	cacheResult, cacheKeyError := cache.RedisClient.Get("account_type:1_outer_cookie:1685491709028360").Result()
	if cacheKeyError != nil {
		panic("获取redis中的cookie值错误" + cacheKeyError.Error())
	}

	cookies := parseCookie(cacheResult)
	const CookieDomain = ".oceanengine.com"

	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
			for i := 0; i < len(cookies); i += 2 {
				err := network.SetCookie(cookies[i], cookies[i+1]).
					WithExpires(&expr).WithDomain(CookieDomain).
					WithSameSite(network.CookieSameSiteNone).
					WithSecure(true).Do(ctx)

				if err != nil {
					log.Println(failMsg + "加载cookie时错误" + err.Error())
					panic(failMsg + "加载cookie时错误" + err.Error())
				}
			}
			return nil
		}),
		chromedp.EmulateViewport(1920, 1080),
		chromedp.Navigate("https://agent.oceanengine.com/admin/company/account/management"),
		chromedp.Sleep(10 * time.Second),
	}); err != nil {
		log.Println(failMsg + "运行浏览器时错误" + err.Error())
		panic(failMsg + "运行浏览器时错误" + err.Error())
	}
	//这里时间长一点，时间短页面可能会出现页面渲染不完整的情况
	//time.Sleep(10 * time.Second)
	//查询是否有指定的dom元素
	ch := addNewTabListener(ctx)

	var ret any
	if err := chromedp.Run(ctx,
		//关闭弹窗、选择点击第二个标签页
		chromedp.Evaluate(fmt.Sprintf(`
	setTimeout(function (){
        var element1 = document.querySelector(".ant-modal-close-x");
        if (element1) {
            element1.click();
        }
    },1500)


    setTimeout(function (){
		var elements = document.querySelectorAll(".ant-tabs-tab");
		if (elements.length >= 2) {
			elements[2].click();
		}
    },2500)
	`),
			&ret),
		//输入框赋值
		chromedp.SendKeys("#adv", CmsMediaAccount.AdvertiserID, chromedp.ByQuery),
		//等待接口响应
		chromedp.Sleep(2*time.Second),
		//搜索
		chromedp.Click(".ant-tabs-tabpane-active .wrapped-form__form__actions button", chromedp.ByQuery),
		//等待接口响应
		chromedp.Sleep(5*time.Second),
		chromedp.Click(".ant-tabs-tabpane-active tr .lock-right button:nth-child(1)", chromedp.ByQuery),
		//进入子账号tab页面、模拟授权,这里等待一下
		chromedp.Sleep(10*time.Second),
	); err != nil {
		log.Println(failMsg + "运行js时错误" + err.Error())
		panic(failMsg + "运行js时错误" + err.Error())
	}

	//切换tab需要创建新的上下文节点
	ctx1, cancel1 := chromedp.NewContext(ctx, chromedp.WithTargetID(<-ch))
	defer cancel1()

	if err := chromedp.Run(ctx1,
		chromedp.Click(".firefly-modal-multi-step-skip", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		//弹窗关闭
		//chromedp.Click(".ovui-modal__footer_extra .flex-row-reverse .oc-button-wrap button", chromedp.ByQuery),
		//chromedp.Click(".ovui-modal__wrap .ovui-modal .ovui-modal__close-icon", chromedp.ByQuery),
		//视图
		chromedp.EmulateViewport(1920, 1080),
		//访问流水页面
		chromedp.Navigate("https://ad.oceanengine.com/cg_trade/finance/flow/list?aadvid="+CmsMediaAccount.AdvertiserID+"&from_app=ad&app_key=0&advid="+CmsMediaAccount.AdvertiserID),
		chromedp.Sleep(10*time.Second),
	); err != nil {
		panic(failMsg + "运行js时错误" + err.Error())
	}

	//等待页面响应
	//time.Sleep(10 * time.Second)

	//判断流水页面dom是否存在
	var flowHeaderCount = -1
	chromedp.Run(ctx1, chromedp.Evaluate(`document.querySelectorAll(".src-pages-flow-list-index-module__content").length`, &flowHeaderCount))
	var flowContentCount = -1
	chromedp.Run(ctx1, chromedp.Evaluate(`document.querySelectorAll(".byted-finance-table").length`, &flowContentCount))
	if flowHeaderCount <= 0 || flowContentCount <= 0 {
		log.Println(failMsg + "字节月度流水页面dom节点不存在")
		panic(failMsg + "字节月度流水页面dom节点不存在")
		return ""
	}

	//运行选择日期的js
	if err := chromedp.Run(ctx1, chromedp.Evaluate(fmt.Sprintf(`
		var ele0 = document.querySelector(".byted-finance-input-inner__wrapper.byted-finance-input-inner__wrapper-border.byted-finance-input-inner__wrapper-size-md.byted-finance-input-inner__wrapper-add-suffix.byted-finance-input-inner__wrapper-filled")
	   if (ele0) {
	       ele0.click()
	   }
	   setTimeout(function (){
	       var targetStartNodes = document.querySelectorAll(".byted-finance-date-date.byted-finance-date-item.byted-finance-date-grid-start");
	       var startDate = targetStartNodes[0];
	       // 如果找到了节点，模拟点击效果
	       if (startDate) {
	           // 触发点击事件
	           startDate.click();
	       }
	   },1000)
	   setTimeout(function (){
	       var targetEndNodes = document.querySelectorAll(".byted-finance-date-date.byted-finance-date-item.byted-finance-date-grid-end");
	       var startEnd = targetEndNodes[0];
	       // 如果找到了节点，模拟点击效果
	       if (startEnd) {
	           // 触发点击事件
	           startEnd.click();
	       }
	   },1000)
	`), &ret),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		log.Println(failMsg + "运行js时错误" + err.Error())
		panic(failMsg + "运行js时错误" + err.Error())
	}
	//等待接口返回数据进行dom渲染
	//time.Sleep(2 * time.Second)
	// 截图
	var screenshot []byte
	if err := chromedp.Run(ctx1,
		chromedp.CaptureScreenshot(&screenshot),
	); err != nil {
		log.Println(failMsg + "全屏截图错误" + err.Error())
		panic(failMsg + "全屏截图错误" + err.Error())
	}

	fileName := CmsMediaAccount.AdvertiserID + ".png"
	dirName := "../static/product_flow_snapshot/" + lastMonth + "/" + strconv.Itoa(int(CmsMediaAccount.ProductID)) + "/"
	// 将截图保存到文件
	if err := os.WriteFile(dirName+fileName, screenshot, 0644); err != nil {
		log.Println(failMsg + err.Error())
		panic(failMsg + "全屏截图错误" + err.Error())
	}
	path, err := utils.UploadLocalFile(dirName+fileName, "data/product_flow_snapshot_detail/"+fileName)
	if err != nil {
		log.Println("上传到tos错误", err)
		panic("上传到tos错误" + err.Error())
	}
	// 截取特定区域的截图
	log.Println("截图保存成功")
	return path
}

// SnapshotKsFlow 快手月流水截图
func (s SnapshotService) SnapshotKsFlow(CmsMediaAccount model.CmsMediaAccount) string {

	lastMonth := utils.GetLastMonth("200601")
	failMsg := fmt.Sprintf("快手月度流水截图错误:月份：%v，产品id：%v，媒体账号id：%v，错误原因：", lastMonth, CmsMediaAccount.ProductID, CmsMediaAccount.AdvertiserID)
	defer func() {
		if r := recover(); r != nil {
			log.Println(failMsg+"%v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	// 创建Chrome无头浏览器选项
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("window-size", "1920,1080"),
	)
	// 初始化浏览器
	ctx, cancel = chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	// 创建Chrome浏览器上下文
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	var redisKey string

	if CmsMediaAccount.BusinessType == "KA" {
		redisKey = "ks_ka_cookie:18623690021"
	} else if CmsMediaAccount.BusinessType == "LA" {
		redisKey = "ks_la_cookie:19922852907"
	}
	cacheResult, cacheKeyError := cache.RedisClient.Get(redisKey).Result()
	if cacheKeyError != nil {
		panic("获取redis中的cookie值错误" + cacheKeyError.Error())
	}

	cookies := parseCookie(cacheResult)
	const CookieDomain = ".kuaishou.com"

	//查询是否有指定的dom元素 ，切换tab需要在最开始打开浏览器tab的地方操作
	ch := addNewTabListener(ctx)

	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.EmulateViewport(1920, 1080),
		chromedp.ActionFunc(func(ctx context.Context) error {
			expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
			for i := 0; i < len(cookies); i += 2 {
				err := network.SetCookie(cookies[i], cookies[i+1]).
					WithExpires(&expr).WithDomain(CookieDomain).
					WithSameSite(network.CookieSameSiteNone).
					WithSecure(true).Do(ctx)

				if err != nil {
					log.Println(failMsg + "加载cookie时错误" + err.Error())
					panic(failMsg + "加载cookie时错误" + err.Error())
				}
			}
			return nil
		}),
		chromedp.Navigate("https://agent.e.kuaishou.com/account/list"),
		chromedp.Sleep(7 * time.Second),
		chromedp.Navigate("https://agent.e.kuaishou.com/account/list"),
		//关闭弹窗
		chromedp.Click(".sc-bYSBpT", chromedp.ByQuery),
		chromedp.Sleep(2 * time.Second),
		//输入要登录的子账号
		chromedp.SendKeys(".agent-row div:nth-child(1) .agent-row .agent-col .flex-start .agent-input-group .agent-input-affix-wrapper input", CmsMediaAccount.AdvertiserID, chromedp.ByQuery),
		chromedp.Sleep(4 * time.Second),
		//访问子账号页面
		chromedp.Click(".agent-table-body tr:nth-child(2) td:nth-child(3) a:nth-child(1)", chromedp.ByQuery),
		//这里时间长一点，时间短页面可能会出现子账号页面渲染不完整的情况
		chromedp.Sleep(7 * time.Second),
	}); err != nil {
		log.Println(failMsg + "运行浏览器时错误" + err.Error())
		panic(failMsg + "运行浏览器时错误" + err.Error())

	}
	//这里时间长一点，时间短页面可能会出现子账号页面渲染不完整的情况
	//time.Sleep(7 * time.Second)

	//切换tab需要创建新的上下文节点
	ctx1, cancel1 := chromedp.NewContext(ctx, chromedp.WithTargetID(<-ch))
	defer cancel1()

	if err := chromedp.Run(ctx1,
		//访问流水页面
		chromedp.Navigate("https://ad.e.kuaishou.com/finance-report/account-flow?__accountId__="+CmsMediaAccount.AdvertiserID+"&__stopPrompt__=true"),
	); err != nil {
		panic(failMsg + "运行js时错误" + err.Error())
	}

	time.Sleep(6 * time.Second)

	//判断流水页面dom是否存在
	var flowHeaderCount = -1
	chromedp.Run(ctx1, chromedp.Evaluate(`document.querySelectorAll(".account-flow__header_title").length`, &flowHeaderCount))
	var flowContentCount = -1
	chromedp.Run(ctx1, chromedp.Evaluate(`document.querySelectorAll(".account-flow__content").length`, &flowContentCount))
	if flowHeaderCount <= 0 || flowContentCount <= 0 {
		log.Println(failMsg + "快手月度流水页面dom节点不存在")
		panic(failMsg + "快手月度流水页面dom节点不存在")
		return ""
	}
	// 移动鼠标到指定位置
	if err := chromedp.Run(ctx1,
		chromedp.MouseEvent("mouseMoved", float64(1732), float64(16)),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		log.Println(failMsg + "移动鼠标时错误" + err.Error())
		panic(failMsg + "移动鼠标时错误" + err.Error())
	}
	//time.Sleep(4 * time.Second)

	var ret any
	if err := chromedp.Run(ctx1, chromedp.Evaluate(fmt.Sprintf(`
	//删除右下角弹窗的dom节点
	var elements = document.querySelectorAll('.ant-popover-content');
	elements.forEach(function(element) {
	  element.remove();
	});
	function getFirstDayOfLastMonth() {
    var today = new Date();

    // 将日期设置为本月的第一天
    today.setDate(0);
    // 减去一天，使日期变为上个月的最后一天
    today.setDate(1);
    var lastDay = today.getDate();
    // 如果需要，你还可以获取上个月的年份和月份
    var lastMonth = today.getMonth(); // 月份是从 0 开始的
    var lastYear = today.getFullYear();
    // 补零函数
    function addLeadingZero(num) {
        return num < 10 ? '0' + num : num;
    }
    return lastYear+'-'+addLeadingZero(lastMonth + 1)+'-'+addLeadingZero(lastDay)
}

function getLastDayOfLastMonth() {
    var today = new Date();
    // 将日期设置为本月的第一天
    today.setDate(1);
    // 减去一天，使日期变为上个月的最后一天
    today.setDate(0);
    var lastDay = today.getDate();
    // 如果需要，你还可以获取上个月的年份和月份
    var lastMonth = today.getMonth(); // 月份是从 0 开始的
    var lastYear = today.getFullYear();
    // 补零函数
    function addLeadingZero(num) {
        return num < 10 ? '0' + num : num;
    }
    return lastYear+'-'+addLeadingZero(lastMonth + 1)+'-'+addLeadingZero(lastDay)
}

var lastDay = getFirstDayOfLastMonth();
var lastDay = getLastDayOfLastMonth();
//点击日期框
var elements0 = document.querySelector('.ant-picker.ant-picker-range');
if(elements0){
	elements0.click()
}
//月份选中上个月
var elements1 = document.querySelector('.ant-picker-header-prev-btn');
if(elements1){
	setTimeout(function (){
		elements1.click()
	},1000)
}
//点击月份第一天
setTimeout(function (){
		var tdElement = document.querySelector('td[title="'+getFirstDayOfLastMonth()+'"]');
		if (tdElement) {
		 	tdElement.click();
		}
    },2000)
//点击月份最后一天
setTimeout(function (){
	var tdElement = document.querySelector('td[title="'+getLastDayOfLastMonth()+'"]');
	if (tdElement) {
	 	tdElement.click();
	} 
},3000)
`), &ret),
		//等待7秒给接口响应时间，这里包含了，异步执行js的时间
		chromedp.Sleep(7*time.Second),
	); err != nil {
		log.Println(failMsg + "加载js时错误" + err.Error())
		panic(failMsg + "加载js时错误" + err.Error())
	}
	//等待7秒给接口响应时间，这里包含了，异步执行js的时间
	//time.Sleep(7 * time.Second)

	// 全屏截图
	var screenshot []byte
	if err := chromedp.Run(ctx1,
		chromedp.CaptureScreenshot(&screenshot),
	); err != nil {
		log.Println(failMsg + "全屏截图错误" + err.Error())
		panic(failMsg + "全屏截图错误" + err.Error())
	}

	fileName := CmsMediaAccount.AdvertiserID + ".png"
	dirName := "../static/product_flow_snapshot/" + lastMonth + "/" + strconv.Itoa(int(CmsMediaAccount.ProductID)) + "/"
	// 将截图保存到文件
	if err := os.WriteFile(dirName+fileName, screenshot, 0644); err != nil {
		log.Println(failMsg + err.Error())
		panic(failMsg + "全屏截图错误" + err.Error())
	}
	path, err := utils.UploadLocalFile(dirName+fileName, "data/product_flow_snapshot_detail/"+fileName)
	if err != nil {
		log.Println("上传到tos错误", err)
		panic("上传到tos错误" + err.Error())
	}
	// 截取特定区域的截图
	log.Println("截图保存成功")
	return path
}

func (s SnapshotService) CronSnapTask(requestData *snapshot.CronTaskRequest) (string, error) {
	now := time.Now()
	snapshotCronLog := "log/cron/product_flow_snapshot/" + now.Format("2006-01-02") + "/"
	//创建cron日志文件夹
	utils2.CreateDir(snapshotCronLog)

	snapshotCronLogOpenFile, snapshotCronLogOpenFileErr := os.OpenFile(snapshotCronLog+"status.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if snapshotCronLogOpenFileErr != nil {
		log.Fatal("创建日志文件失败")
	}
	defer snapshotCronLogOpenFile.Close()
	log.SetOutput(snapshotCronLogOpenFile)
	log.Println("任务开始执行")

	cacheResult, _ := cache.RedisClient.Get("cron-crm-lock-" + now.Format("2006-01-02")).Result()

	if cacheResult == "1" {
		log.Println("脚本正在运行中")
		return "", fmt.Errorf("脚本正在运行中")
	}

	//设置脚本锁 20小时执行时间
	cache.RedisClient.Set("cron-crm-lock-"+now.Format("2006-01-02"), 1, 72000*1000*1000*1000)

	//当前日日天数
	var productConfigList []model.CrmProductFlowSnapshotConfig
	database.CrmDB.Where("send_date = ? and status = 1", requestData.CronDate).Find(&productConfigList)

	if len(productConfigList) <= 0 {
		log.Println("未查到有效的产品流水配置")
	}

	lastMonth := utils.GetLastMonth("200601")

	for i := range productConfigList {
		dirName := "log/cron/product_flow_snapshot/" + lastMonth + "/"
		//创建日志文件夹
		utils2.CreateDir(dirName)
		//设置日志文件
		openFile, openFileErr := os.OpenFile(dirName+"product_config_"+strconv.Itoa(int(productConfigList[i].ID))+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if openFileErr != nil {
			log.Fatal("创建日志文件失败")
		}
		log.SetOutput(openFile)
		//查询用户
		var crmUser = &model.CrmUser{}
		resultErr := database.CrmDB.Where("id = ? and state = 1", productConfigList[i].CreateUserID).First(&crmUser)

		var configMsg string
		configMsg = "月度流水配置id：" + strconv.Itoa(int(productConfigList[i].ID))

		//已离职或者账号已经停用不再发送
		if resultErr != nil && errors.Is(resultErr.Error, gorm.ErrRecordNotFound) {
			log.Println(configMsg + "对应的创建人账号不存在或者已经停用")
			continue
		}
		//查询关联的产品
		productIds, _ := utils.ConvertStringToIntSlice(productConfigList[i].ProductIds)
		var cmsProducts []model.CmsProduct
		database.CmsDB.Where("id in ?", productIds).Find(&cmsProducts)
		if len(cmsProducts) < 0 {
			log.Println(configMsg + "未查到到关联产品")

			continue
		}
		//循环产品
		for j := range cmsProducts {
			logFile := strconv.Itoa(int(cmsProducts[j].ID)) + ".log"
			productOpenFile, productOpenFileErr := os.OpenFile(dirName+logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if productOpenFileErr != nil {
				log.Println(configMsg + "产品id" + strconv.Itoa(int(cmsProducts[j].ID)) + "创建日志文件失败")
				continue
			}
			log.SetOutput(productOpenFile)
			defer productOpenFile.Close()

			//创建文截图保存文件夹
			flowSnapDirName := "../static/product_flow_snapshot/" + lastMonth + "/" + strconv.Itoa(int(cmsProducts[j].ID))
			err := os.MkdirAll(flowSnapDirName, 0755)
			if err != nil {
				log.Printf("创建文件夹"+flowSnapDirName+"出错:", err)
				continue
			}

			//查询产品关联的媒体账号
			var cmsMediaAccounts []model.CmsMediaAccount
			database.CmsDB.Where("product_id = ?", cmsProducts[j].ID).Find(&cmsMediaAccounts)
			if len(cmsMediaAccounts) == 0 {
				log.Printf("产品id：%v无关联媒体账号", cmsProducts[j].ID)
				continue
			}

			flowImageNum := 0
			//循环媒体账号
			for k := range cmsMediaAccounts {
				var productMonth = &model.CmsReportCustomProductMonth{}
				//查询产品关联的媒体账号是否有流水
				productMonthErr := database.MapiReportDB.Where("product_id = ? and accept_month = ? and media_type = ? and advertiser_id = ? ", cmsMediaAccounts[k].ProductID, lastMonth, cmsMediaAccounts[k].MediaID, cmsMediaAccounts[k].AdvertiserID).First(&productMonth)
				//无流水记录
				if productMonthErr != nil && errors.Is(productMonthErr.Error, gorm.ErrRecordNotFound) {
					log.Printf("产品名称，%v产品id%v，不存在流水记录\n", cmsProducts[j].ProductName, cmsProducts[j].ID)
					continue
				}

				//流水金额小于等于0.00 直接跳过
				if productMonth.Cost <= 0.00 {
					log.Printf("产品名称，%v产品id%v，流水金额小于0.00\n", cmsProducts[j].ProductName, cmsProducts[j].ID)

					continue
				}
				var path string
				//快手
				if cmsMediaAccounts[k].MediaID == 1 {
					path = s.SnapshotKsFlow(cmsMediaAccounts[k])
				}
				//字节
				if cmsMediaAccounts[k].MediaID == 2 {
					path = s.SnapshotZjFlow(cmsMediaAccounts[k])
				}

				if path != "" {
					crmProductFlowSnapshotDetail := model.CrmProductFlowSnapshotDetail{
						Month:        lastMonth,
						MediaID:      cmsMediaAccounts[k].MediaID,
						ProductID:    cmsMediaAccounts[k].ProductID,
						AdvertiserID: cmsMediaAccounts[k].AdvertiserID,
						SnapShotURL:  path,
					}
					database.CrmDB.Create(&crmProductFlowSnapshotDetail)
					flowImageNum++
				}

			}
			_, statErr := os.Stat(flowSnapDirName)

			if flowImageNum > 0 && statErr == nil {
				//压缩文件地址
				now := time.Now()
				_, err := utils.CompressFolderZip(flowSnapDirName, flowSnapDirName+".zip")
				if err != nil {
					log.Println("压缩文件错误", err)
				}

				//上传到tos
				path, err := utils.UploadLocalFile(flowSnapDirName+".zip", "data/product_month_flow_snap/"+lastMonth+"/"+strconv.Itoa(int(cmsProducts[j].ID))+"_"+strconv.Itoa(int(now.Unix()))+".zip")
				if err != nil {
					log.Println("上传到tos错误", err)
					continue
				}
				log.Println("压缩包文件上传成功,保存tos路径", path)

				//删除文件夹
				os.RemoveAll(flowSnapDirName)
				_, err = os.Stat(flowSnapDirName + ".zip")
				if err == nil {
					//删除zip文件
					err = os.Remove(flowSnapDirName + ".zip")
				}
				var requestData = make(map[string]interface{})
				requestData["create_user_id"] = productConfigList[i].CreateUserID
				requestData["product_id"] = cmsProducts[j].ID
				requestData["product_name"] = cmsProducts[j].ProductName
				requestData["month"] = lastMonth
				requestData["zip_url"] = path
				//发送压缩包地址
				requestData["notify_type"] = 2

				//写入crm_product_flow_snapshot表
				crmProductFlowSnapshot := model.CrmProductFlowSnapshot{
					ProductFlowSnapshotConfigID: productConfigList[i].ID,
					ProductID:                   cmsProducts[j].ID,
					Month:                       lastMonth,
					ZipURL:                      path,
				}
				database.CrmDB.Create(&crmProductFlowSnapshot)
				//通知php 发送压缩包地址、配置人
				library.SendPOSTRequest("http://127.0.0.1:8000/external/screenshot/notify", requestData)
			}
		}
	}
	log.SetOutput(snapshotCronLogOpenFile)
	log.Println("任务结束执行")

	//脚本解锁
	cache.RedisClient.Del("cron-crm-lock-" + now.Format("2006-01-02"))
	return "", nil
}

func (s SnapshotService) UpdateCrmScreenshotJob(jobNo string, status int, url string, failMsg string) (string, error) {

	crmScreenshotJob := &model.CrmScreenshotJob{}

	result := database.CrmDB.Where("job_no = ?", jobNo).First(&crmScreenshotJob)

	if result.Error != nil && errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return "fail", fmt.Errorf("未找到记录")
	}
	//修改昵称
	crmScreenshotJob.Status = status
	if len(url) > 0 {
		crmScreenshotJob.URL = url
	}
	if len(failMsg) > 0 {
		crmScreenshotJob.FailMsg = failMsg
	}

	database.CrmDB.Save(crmScreenshotJob)

	return "ok", nil
}

func parseCookie(c string) []string {
	cookieArr := strings.Split(c, ";")
	var mapCookies = map[string]string{}
	for _, cookie := range cookieArr {
		pairs := strings.Split(strings.TrimSpace(cookie), "=")
		if len(pairs) == 2 {
			mapCookies[pairs[0]] = pairs[1]
		}
	}
	var cookies []string
	for k, v := range mapCookies {
		cookies = append(cookies, k, v)
	}
	return cookies
}

/**
 * 注册新tab标签的监听服务
 */
func addNewTabListener(ctx context.Context) <-chan target.ID {
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	return chromedp.WaitNewTarget(ctx, func(info *target.Info) bool {
		return info.URL != ""
	})
}
