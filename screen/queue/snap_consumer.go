package queue

import (
	"context"
	"crm.com/common/cache"
	"crm.com/common/database"
	utils2 "crm.com/common/utils"
	"crm.com/screen/api/model"
	"crm.com/screen/api/utils"
	"crm.com/screen/api/validator/snapshot"
	"crm.com/screen/library"
	"encoding/json"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/streadway/amqp"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// CustomError 是一个自定义错误类型
type CustomError struct {
	Message string
}

// 实现 error 接口的 Error 方法
func (e CustomError) Error() string {
	return e.Message
}

// SnapshotZjConsumer 字节充值、转款、退款截图
func SnapshotZjConsumer(d amqp.Delivery) {
	defer d.Ack(false)

	var TaskMsg string
	var DoTaskRequest snapshot.DoTaskRequest
	_ = json.Unmarshal(d.Body, &DoTaskRequest)

	defer func() {
		if r := recover(); r != nil {
			database.CrmDB.Model(&model.CrmScreenshotJob{}).Where("job_no = ?", DoTaskRequest.JobNo).Updates(map[string]interface{}{"status": 3, "fail_msg": fmt.Sprintf("Panic recovered: %v", r)})
		}
	}()

	TaskMsg = "截图任务编号" + DoTaskRequest.JobNo

	now := time.Now()
	// 使用当前时间获取今天的日期
	today := now.Format("2006-01-02")
	dirName := "log/snapshot_consumer/" + today + "/"

	//创建日志文件夹
	utils2.CreateDir(dirName)
	//设置日志文件
	openFile, openFileErr := os.OpenFile(dirName+DoTaskRequest.JobNo+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if openFileErr != nil {
		panic(TaskMsg + "创建日志文件失败")
	}
	log.SetOutput(openFile)
	defer openFile.Close()

	log.Println("zj消费了,消息内容为", string(d.Body))

	crmScreenshotJob := &model.CrmScreenshotJob{}
	database.CrmDB.Model(&model.CrmScreenshotJob{}).Where("job_no = ?", DoTaskRequest.JobNo).First(&crmScreenshotJob)
	if crmScreenshotJob.Status == 2 {
		log.Println(TaskMsg + "任务状态已完成")
		return
	}

	database.CrmDB.Model(&model.CrmScreenshotJob{}).Where("job_no = ?", DoTaskRequest.JobNo).Updates(map[string]interface{}{"status": 1, "fail_msg": ""})

	ctx1, cancel1 := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel1()

	// 创建Chrome无头浏览器选项
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("window-size", "1920,1080"),
	)
	// 初始化浏览器
	ctx2, cancel2 := chromedp.NewExecAllocator(ctx1, opts...)
	defer cancel2()
	// 创建Chrome浏览器上下文
	ctx3, cancel3 := chromedp.NewContext(ctx2)
	defer cancel3()

	cacheResult, cacheKeyError := cache.RedisClient.Get("account_type:1_outer_cookie:1685491709028360").Result()
	if cacheKeyError != nil {
		panic(TaskMsg + "获取redis中的cookie值错误" + cacheKeyError.Error())
	}

	cookies := parseCookie(cacheResult)
	const CookieDomain = ".oceanengine.com"

	if err := chromedp.Run(ctx3, chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
			for i := 0; i < len(cookies); i += 2 {
				err := network.SetCookie(cookies[i], cookies[i+1]).
					WithExpires(&expr).WithDomain(CookieDomain).
					WithSameSite(network.CookieSameSiteNone).
					WithSecure(true).Do(ctx)
				if err != nil {
					panic(err)
				}
			}
			return nil
		}),
		chromedp.Navigate("https://agent.oceanengine.com/admin/fundModule/flowQuery/transferRecord"),
		//chromedp.WaitVisible(".okee-component-search-table-container", chromedp.ByQuery),
		//字节表格是异步加载的，需要等待的时间稍微长一点，等待30秒页面加载完成效果最佳
		chromedp.Sleep(30 * time.Second),
	}); err != nil {
		log.Println(TaskMsg+"访问链接错误", err)
	}

	var TableResCount = -1
	chromedp.Run(ctx3, chromedp.Evaluate(`document.querySelectorAll(".byted-card-body").length`, &TableResCount))
	if TableResCount <= 0 {
		log.Println(TaskMsg + "异步dom表格未找到")
		panic(TaskMsg + "异步dom表格未找到")
	}

	var ret any
	if err := chromedp.Run(ctx3, chromedp.Evaluate(fmt.Sprintf(`
		//关闭弹窗
	  var element1 = document.querySelector(".ant-modal-close-x");
	  if(element1){
			element1.click();
		}
	
		//分页
		var ele = document.querySelector(".byted-input-inner__wrapper.byted-input-inner__wrapper-border.byted-input-inner__wrapper-size-xs.byted-input-inner__wrapper-add-suffix.byted-input-inner__wrapper-filled")
		if(ele){
			ele.click()
		}
	
		//分页的数量
	  setTimeout(function() {
	      var targetNodes = document.querySelectorAll(".byted-list-item-container.byted-list-item-container-size-xs.byted-list-item-container-level-1.byted-select-option-container");
	      var fifthNode = targetNodes[4];
	
	      // 如果找到了节点，模拟点击效果
	      if (fifthNode) {
	          // 触发点击事件
	          fifthNode.click();
	      } else {
	          console.error('未找到符合条件的第5个节点');
	      }
	  }, 2000);
	
	`), &ret),
		//设置100条分页后页面需要加载时间
		chromedp.Sleep(10*time.Second),
	); err != nil {
		log.Println(TaskMsg+"运行弹窗js错误", err)
		panic(TaskMsg + "运行弹窗js错误" + err.Error())
	}
	//设置100条分页后页面需要加载时间
	//time.Sleep(10 * time.Second)

	// 遍历切片，在每个元素两侧添加引号
	for i, str := range DoTaskRequest.AdvertiserIds {
		DoTaskRequest.AdvertiserIds[i] = `"` + str + `"`
	}

	// 使用 strings.Join 将切片转换为以逗号分隔的字符串
	resultAdvertiserIds := strings.Join(DoTaskRequest.AdvertiserIds, ",")

	if err := chromedp.Run(ctx3, chromedp.Evaluate(fmt.Sprintf(`
	 	//运行俊超的js
			// 脚本方法定义
	const type = '头条';
	
	const config = {
	头条: {
	 idColumnTitle: '转入方账户', //包含 ID 信息的列名称
	 idColumnTitle2: '转出方账户', //包含 ID 信息的列名称(第二种情况)
	 domClassName: '.okee-component-col-name-id-combo__id',
	 domTitleClassName: '.byted-Table-HeadCellContentTitle',
	 needScreenshotAdIds: [`+resultAdvertiserIds+`], //本地需要截图的媒体账号数据
	 needSaveColumnTitle: [
	   '转账时间',
	   '转出方账户',
	   '转入方账户',
	   '总金额(元)',
	   '转账类型',
	   // '操作人',
	 ], // 需要保留的列 白名单
	 timeColumnTitle: '转账时间',
	 startTime: '`+DoTaskRequest.BeginTime+`',
	 endTime: '`+DoTaskRequest.EndTime+`',
	 screenHotDomName: 'byted-Table-Frame',
	},
	快手: {
	 idColumnTitle: '账户ID', //包含 ID 信息的列名称
	 idColumnTitle2: '不需要第二列', //包含 ID 信息的列名称(第二种情况)
	 domClassName: '',
	 domTitleClassName: '',
	 needScreenshotAdIds: [
	   '12862624',
	   '12863235',
	   '27697890',
	   '25995693',
	   '25320908',
	   '27468200',
	   '25359088',
	   '25584089',
	 ], //本地需要截图的媒体账号数据
	 needSaveColumnTitle: ['操作类型', '账户ID', '日期', '总金额(元)'], // 需要保留的列 白名单
	 timeColumnTitle: '日期',
	 startTime: '2024-03-12 10:02:10',
	 endTime: '2024-03-12 10:46:04',
	 screenHotDomName: 'agent-table-fixed-column',
	},
	};
	
	init();
	// 1、初始化执行
	function init() {
	filterTableRowsDom();
	hideColumnsAndCellsNotInWhitelist();
	}
	// 2、过滤 Table 内容行
	function filterTableRowsDom() {
	// 第一步 拿到各类的数据标识的位置下标
	//  拿到账户 ID下标
	var thIndex = findThIndexByContent(config[type].idColumnTitle); // 查找目标ID列位置 第一种 （转入）
	var thIndex2 = findThIndexByContent(config[type].idColumnTitle2); // 查找目标ID列位置 第二种 （转出）
	//  只展示操作人为 OPENAPI 的数据
	var operateTypeIndex = findThIndexByContent('操作人'); // 查找操作人列位置
	//  拿到时间那一列
	var timeIndex = findThIndexByContent(config[type].timeColumnTitle); // 查找时间列位置
	console.log(
	 '匹配到的下标为: //ID转入：' +
	   thIndex +
	   '//ID转出：' +
	   thIndex2 +
	   '//操作人:' +
	   operateTypeIndex +
	   '//时间:' +
	   timeIndex
	);
	// 第二步  过滤表格内容
	hideRowsNotInWhitelist(
	 thIndex,
	 thIndex2,
	 operateTypeIndex,
	 config[type].needScreenshotAdIds,
	 timeIndex
	);
	}
	
	// 3、过滤表格内容列
	
	// 通用方法定义：根据列名称找到目标列下标
	function findThIndexByContent(thContent) {
	var table = document.querySelector('table');
	if (!table) {
	 console.log('未找到指定的表格元素');
	 return -1;
	}
	
	var thead = table.querySelector('thead');
	if (!thead) {
	 console.log('未找到表格的表头元素');
	 return -1;
	}
	
	var thList = thead.querySelectorAll('th');
	for (var i = 0; i < thList.length; i++) {
	 var th = thList[i];
	 if (th.textContent.trim() === thContent) {
	   return i;
	 }
	}
	
	console.log('未找到匹配的 th 标签');
	return -1;
	}
	
	// 隐藏行
	function hideRowsNotInWhitelist(
	thIndex,
	thIndex2,
	operateTypeIndex,
	whitelist,
	timeIndex
	) {
	var tbody = document.querySelector('tbody');
	if (!tbody) {
	 console.log('未找到表格的 tbody 元素');
	 return;
	}
	
	var rows = tbody.querySelectorAll('tr');
	for (var i = 0; i < rows.length; i++) {
	 var row = rows[i];
	 var tds = row.querySelectorAll('td');
	 if (tds.length < thIndex + 1) {
	   continue; // 忽略没有目标td 的行
	 }
	 // 判断是否为 OPENAPI 执行过滤
	 var operateDom = tds[operateTypeIndex];
	 if (type == '头条' && operateDom.textContent.trim() !== 'OPENAPI') {
	   row.remove(); // 删除
	   continue; // 忽略没有目标td 的行
	 }
	
	 // 判断是否在时间范围内
	 var timeDom = tds[timeIndex];
	 if (timeDom && timeDom.textContent.trim()) {
	   let currentRowTime = timeDom.textContent.trim();
	   console.log(currentRowTime);
	   if (
	     new Date(currentRowTime) < new Date(config[type].startTime) ||
	     new Date(currentRowTime) > new Date(config[type].endTime)
	   ) {
	     row.remove(); // 删除
	     continue; // 忽略没有在时间范围的行
	   }
	 }
	
	 // 第一个转入列内容 寻找可匹配的ID
	 var secondTd = tds[thIndex];
	 var comboIdElement = null;
	
	 // 快手和头条分开处理，快手此处无类名
	 if (!config[type].domClassName) {
	   comboIdElement = secondTd;
	 } else {
	   comboIdElement = secondTd.querySelector(config[type].domClassName);
	 }
	 if (!comboIdElement) {
	   continue; // 忽略没有目标类名的元素
	 }
	
	 var comboId = comboIdElement.textContent.trim();
	 var numericValue = comboId.replace(/\D/g, ''); // 保留数字内容
	
	 if (!whitelist.includes(numericValue)) {
	   // 未匹配到，则第二个转出的列内容继续找 如果找不到则进行删除
	   if (thIndex2 !== -1) {
	     let secondTd2 = tds[thIndex2];
	     var comboIdElement2 = null;
	     // 快手和头条分开处理，快手此处无类名
	     if (!config[type].domClassName) {
	       comboIdElement2 = secondTd;
	     } else {
	       comboIdElement2 = secondTd2.querySelector(config[type].domClassName);
	     }
	
	     if (!comboIdElement2) {
	       continue; // 忽略没有目标类名的元素
	     }
	
	     var comboId2 = comboIdElement2.textContent.trim();
	     var numericValue2 = comboId2.replace(/\D/g, ''); // 保留数字内容
	
	     if (!whitelist.includes(numericValue2)) {
	       row.remove(); // 隐藏不在白名单内的行
	     }
	   } else {
	     row.remove(); // 隐藏不在白名单内的行
	   }
	 }
	}
	}
	
	// 隐藏列
	function hideColumnsAndCellsNotInWhitelist() {
	var thead = document.querySelector('thead');
	if (!thead) {
	 console.log('未找到表格的 thead 元素');
	 return;
	}
	
	var thList = thead.querySelectorAll('th');
	for (var i = 0; i < thList.length; i++) {
	 var th = thList[i];
	
	 var contentTitle = null;
	 if (config[type].domTitleClassName) {
	   contentTitle = th.querySelector(config[type].domTitleClassName);
	 } else {
	   contentTitle = th;
	 }
	 if (!contentTitle) {
	   continue; // 忽略没有目标类名的元素
	 }
	
	 var content = contentTitle.textContent.trim();
	 if (
	   !config[type].needSaveColumnTitle.includes(content) &&
	   !content.includes('总金额')
	 ) {
	   th.style.display = 'none'; // 隐藏不在白名单内的 th 标签
	
	   var tbody = document.querySelector('tbody');
	   if (!tbody) {-
	     console.log('未找到表格的 tbody 元素');
	     return;
	   }
	
	   var rows = tbody.querySelectorAll('tr');
	   for (var j = 0; j < rows.length; j++) {
	     var tdList = rows[j].querySelectorAll('td');
	     if (tdList.length > i) {
	       tdList[i].style.display = 'none'; // 隐藏对应下标的 td 标签
	     }
	   }
	 } else {
	   // 处理快手自适应  每个 td都要均分宽度
	   if (type == '快手') {
	     var tbody = document.querySelector('tbody');
	     var rows = tbody.querySelectorAll('tr');
	     for (var j = 0; j < rows.length; j++) {
	       var tdList = rows[j].querySelectorAll('td');
	       if (tdList.length > i) {
	         tdList[i].style.width = "25%"; // 隐藏对应下标的 td 标签
	       }
	     }
	   }
	 }
	}
	
	// 列头宽度优化
	var colgroupAll = document.querySelectorAll('colgroup');
	colgroupAll.forEach((element) => {
	 element.remove();
	});
	if (type == '头条') {
	 var tableAll = document.querySelectorAll('table');
	 tableAll.forEach((ele) => {
	   ele.style.minWidth = 'auto';
	 });
	} else {
	 // 自适应逻辑已在过滤阶段处理
	 // end
	}
	}
	`), &ret),
		chromedp.Sleep(3*time.Second),
	); err != nil {
		panic(TaskMsg + "运行过滤dom的js脚本错误" + err.Error())
	}
	//给予过滤数据的js脚本一定运行时间
	//time.Sleep(3 * time.Second)

	var resCount = -1
	chromedp.Run(ctx3, chromedp.Evaluate(`document.querySelectorAll(".byted-Table-Implement tr").length`, &resCount))
	log.Println(TaskMsg+"tr的数量是", resCount)

	if resCount > 1 {
		var spcHeight int64 = 1080
		//表格区域截图
		var screenshot []byte
		if resCount > 9 {
			//>9是因为，屏幕最多显示9行， 65是tr的高度。截图需要更改视图的高度 resCount + 1是因为表头占据了一个65的高度
			spcHeight = int64((resCount + 1) * 65)
		}
		if err := chromedp.Run(ctx3,
			//.byted-Table-Frame.byted-Table-Frame_theadFixed
			chromedp.EmulateViewport(1920, spcHeight),
			chromedp.Screenshot(".byted-Table-Frame", &screenshot, chromedp.ByQuery)); err != nil {
			log.Println(TaskMsg+"截图错误", err)
			panic(TaskMsg + "截图错误" + err.Error())
		}

		//全屏截图 可观察分页设置情况
		//var screenshot []byte
		//if err := chromedp.Run(ctx,
		//	chromedp.CaptureScreenshot(&screenshot),
		//); err != nil {
		//	log.Fatal(err)
		//}
		//创建日志文件夹
		utils2.CreateDir("../static/zj_screen_snap/")
		now := time.Now()
		screenshotImageName := DoTaskRequest.JobNo + "_" + strconv.Itoa(int(now.Unix())) + ".png"
		// 将截图保存到文件
		if err := ioutil.WriteFile("../static/zj_screen_snap/"+screenshotImageName, screenshot, 0644); err != nil {
			log.Println(TaskMsg+"文件保存错误", err)
			panic(TaskMsg + "文件保存错误" + err.Error())
		}
		path, err := utils.UploadLocalFile("../static/zj_screen_snap/"+screenshotImageName, "data/zj_screen_snap/"+screenshotImageName)
		if err != nil {
			log.Println(TaskMsg+"上传到tos错误", err)
			panic(TaskMsg + "上传到tos错误" + err.Error())
		}
		log.Println(TaskMsg+"截图保存成功,保存tos路径", path)

		//修改任务状态
		database.CrmDB.Model(crmScreenshotJob).Where("job_no = ?", DoTaskRequest.JobNo).Updates(map[string]interface{}{"status": 2, "url": path, "fail_msg": ""})

		//删除截图文件
		err = os.Remove("../static/zj_screen_snap/" + screenshotImageName)
		if err != nil {
			log.Println(TaskMsg+"删除系统本地截图文件错误", err)
		}

		var requestData = make(map[string]interface{})
		requestData["job_no"] = DoTaskRequest.JobNo
		requestData["image_url"] = path
		requestData["notify_type"] = 1
		log.Println(TaskMsg+"请求php参数", requestData)

		//通知php给审批单发起人发送截图  http://10.200.16.50:9055/external/screenshot/notify
		library.SendPOSTRequest("http://127.0.0.1:8000/external/screenshot/notify", requestData)
	} else {
		panic(TaskMsg + "获取到的tr数量小于1")
	}
}

// SnapshotKSConsumer 快手充值、转款、退款截图
func SnapshotKSConsumer(d amqp.Delivery) {
	defer d.Ack(false)

	var TaskMsg string
	var DoTaskRequest snapshot.DoTaskRequest
	_ = json.Unmarshal(d.Body, &DoTaskRequest)

	defer func() {
		if r := recover(); r != nil {
			database.CrmDB.Model(&model.CrmScreenshotJob{}).Where("job_no = ?", DoTaskRequest.JobNo).Updates(map[string]interface{}{"status": 3, "fail_msg": fmt.Sprintf("Panic recovered: %v", r)})
		}
	}()

	TaskMsg = "截图任务编号" + DoTaskRequest.JobNo
	now := time.Now()
	// 使用当前时间获取今天的日期
	today := now.Format("2006-01-02")
	dirName := "log/snapshot_consumer/" + today + "/"

	//创建日志文件夹
	utils2.CreateDir(dirName)
	//设置日志文件
	openFile, openFileErr := os.OpenFile(dirName+DoTaskRequest.JobNo+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if openFileErr != nil {
		log.Println(TaskMsg + "创建日志文件失败" + openFileErr.Error())
		panic(TaskMsg + "创建日志文件失败" + openFileErr.Error())
	}
	log.SetOutput(openFile)
	defer openFile.Close()

	database.CrmDB.Model(&model.CrmScreenshotJob{}).Where("job_no = ?", DoTaskRequest.JobNo).Updates(map[string]interface{}{"status": 1, "fail_msg": ""})

	ctx1, cancel1 := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel1()
	// 创建Chrome无头浏览器选项
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("window-size", "1920,1080"),
	)
	// 初始化浏览器
	ctx2, cancel2 := chromedp.NewExecAllocator(ctx1, opts...)
	defer cancel2()

	// 创建Chrome浏览器上下文
	ctx3, cancel3 := chromedp.NewContext(ctx2)
	defer cancel3()

	var redisKey string

	if DoTaskRequest.BusinessType == "KA" {
		redisKey = "ks_ka_cookie:18623690021"
	} else if DoTaskRequest.BusinessType == "LA" {
		redisKey = "ks_la_cookie:19922852907"
	}
	cacheResult, cacheKeyError := cache.RedisClient.Get(redisKey).Result()
	if cacheKeyError != nil {
		log.Println(TaskMsg + "获取redis中的cookie值错误" + cacheKeyError.Error())
		panic(TaskMsg + "获取redis中的cookie值错误" + cacheKeyError.Error())
	}

	cookies := parseCookie(cacheResult)
	const CookieDomain = ".kuaishou.com"

	if err := chromedp.Run(ctx3, chromedp.Tasks{
		chromedp.EmulateViewport(1920, 1080),
		chromedp.ActionFunc(func(ctx context.Context) error {
			expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
			for i := 0; i < len(cookies); i += 2 {
				err := network.SetCookie(cookies[i], cookies[i+1]).
					WithExpires(&expr).WithDomain(CookieDomain).
					WithSameSite(network.CookieSameSiteNone).
					WithSecure(true).Do(ctx)
				if err != nil {
					log.Println(TaskMsg + "加载cookie时错误" + err.Error())
					panic(TaskMsg + "加载cookie时错误" + err.Error())
				}
			}
			return nil
		}),
		//访问账号列表页面
		chromedp.Navigate("https://agent.e.kuaishou.com/finance-management/record/advertiser"),

		// 等待页面加载完，这里给足够的时间来渲染
		chromedp.Sleep(7 * time.Second),
		chromedp.Navigate("https://agent.e.kuaishou.com/finance-management/record/advertiser"),
		// 等待页面加载完，这里给足够的时间来渲染
		chromedp.Sleep(10 * time.Second),
	}); err != nil {
		log.Println(TaskMsg + "运行浏览器时错误" + err.Error())
		panic(TaskMsg + "运行浏览器时错误" + err.Error())
	}

	var resCount = -1
	//表格区域存在
	chromedp.Run(ctx3, chromedp.Evaluate(`document.querySelectorAll(".agent-table-tbody").length`, &resCount))
	log.Println(TaskMsg+"表格元素dom节点数量", resCount)

	if resCount > 0 {
		var ret any
		if err := chromedp.Run(ctx3, chromedp.Evaluate(fmt.Sprintf(`
	var element1 = document.querySelector(".iLwFbl");
    if(element1){
		element1.click();
	}
	//优化点，找到所有弹窗
	var elements = document.querySelectorAll('.agent-popover.message-box-list.agent-popover-placement-bottomRight');
	if(elements){
		// 删除弹窗
		elements.forEach(function(element) {
		   element.remove();
		});
	}
`), &ret)); err != nil {
			log.Println(TaskMsg + "执行关闭弹窗js错误" + err.Error())
			panic(TaskMsg + "执行关闭弹窗js错误" + err.Error())
		}

		// 遍历切片，在每个元素两侧添加引号
		for i, str := range DoTaskRequest.AdvertiserIds {
			DoTaskRequest.AdvertiserIds[i] = `"` + str + `"`
		}

		// 使用 strings.Join 将切片转换为以逗号分隔的字符串
		resultAdvertiserIds := strings.Join(DoTaskRequest.AdvertiserIds, ",")

		if err := chromedp.Run(ctx3, chromedp.Evaluate(fmt.Sprintf(`
    	//运行俊超的js
		// 脚本方法定义
// 脚本方法定义
const type = '快手';

const config = {
  头条: {
    idColumnTitle: '转入方账户', //包含 ID 信息的列名称
    idColumnTitle2: '转出方账户', //包含 ID 信息的列名称(第二种情况)
    domClassName: '.okee-component-col-name-id-combo__id',
    domTitleClassName: '.byted-Table-HeadCellContentTitle',
    needScreenshotAdIds: ['1741311215191117'], //本地需要截图的媒体账号数据
    needSaveColumnTitle: [
      '转账时间',
      '转出方账户',
      '转入方账户',
      '总金额(元)',
      '转账类型',
      // '操作人',
    ], // 需要保留的列 白名单
    timeColumnTitle: '转账时间',
    startTime: '2024-03-12 14:57:59',
    endTime: '2024-03-12 14:58:35',
    screenHotDomName: 'byted-Table-Frame',
  },
  快手: {
    idColumnTitle: '账户ID', //包含 ID 信息的列名称
    idColumnTitle2: '不需要第二列', //包含 ID 信息的列名称(第二种情况)
    domClassName: '',
    domTitleClassName: '',
    needScreenshotAdIds: [
  `+resultAdvertiserIds+`
    ], //本地需要截图的媒体账号数据
    needSaveColumnTitle: ['操作类型', '账户ID', '日期', '总金额(元)'], // 需要保留的列 白名单
    timeColumnTitle: '日期',
    startTime: '`+DoTaskRequest.BeginTime+`',
    endTime:'`+DoTaskRequest.EndTime+`',
    screenHotDomName: 'agent-table-fixed-column',
  },
};

init();
// 1、初始化执行
function init() {
  filterTableRowsDom();
  hideColumnsAndCellsNotInWhitelist();
}
// 2、过滤 Table 内容行
function filterTableRowsDom() {
  // 第一步 拿到各类的数据标识的位置下标
  //  拿到账户 ID下标
  var thIndex = findThIndexByContent(config[type].idColumnTitle); // 查找目标ID列位置 第一种 （转入）
  var thIndex2 = findThIndexByContent(config[type].idColumnTitle2); // 查找目标ID列位置 第二种 （转出）
  //  只展示操作人为 OPENAPI 的数据
  var operateTypeIndex = findThIndexByContent('操作人'); // 查找操作人列位置
  //  拿到时间那一列
  var timeIndex = findThIndexByContent(config[type].timeColumnTitle); // 查找时间列位置
  console.log(
    '匹配到的下标为: //ID转入：' +
      thIndex +
      '//ID转出：' +
      thIndex2 +
      '//操作人:' +
      operateTypeIndex +
      '//时间:' +
      timeIndex
  );
  // 第二步  过滤表格内容
  hideRowsNotInWhitelist(
    thIndex,
    thIndex2,
    operateTypeIndex,
    config[type].needScreenshotAdIds,
    timeIndex
  );
}

// 3、过滤表格内容列

// 通用方法定义：根据列名称找到目标列下标
function findThIndexByContent(thContent) {
  var table = document.querySelector('table');
  if (!table) {
    console.log('未找到指定的表格元素');
    return -1;
  }

  var thead = table.querySelector('thead');
  if (!thead) {
    console.log('未找到表格的表头元素');
    return -1;
  }

  var thList = thead.querySelectorAll('th');
  for (var i = 0; i < thList.length; i++) {
    var th = thList[i];
    if (th.textContent.trim() === thContent) {
      return i;
    }
  }

  console.log('未找到匹配的 th 标签');
  return -1;
}

// 隐藏行
function hideRowsNotInWhitelist(
  thIndex,
  thIndex2,
  operateTypeIndex,
  whitelist,
  timeIndex
) {
  var tbody = document.querySelector('tbody');
  if (!tbody) {
    console.log('未找到表格的 tbody 元素');
    return;
  }

  var rows = tbody.querySelectorAll('tr');
  for (var i = 0; i < rows.length; i++) {
    var row = rows[i];
    var tds = row.querySelectorAll('td');
    if (tds.length < thIndex + 1) {
      continue; // 忽略没有目标td 的行
    }
    // 判断是否为 OPENAPI 执行过滤
    var operateDom = tds[operateTypeIndex];
    if (type == '头条' && operateDom.textContent.trim() !== 'OPENAPI') {
      row.remove(); // 删除
      continue; // 忽略没有目标td 的行
    }

    // 判断是否在时间范围内
    var timeDom = tds[timeIndex];
    if (timeDom && timeDom.textContent.trim()) {
      let currentRowTime = timeDom.textContent.trim();
      console.log(currentRowTime);
      if (
        new Date(currentRowTime) < new Date(config[type].startTime) ||
        new Date(currentRowTime) > new Date(config[type].endTime)
      ) {
        row.remove(); // 删除
        continue; // 忽略没有在时间范围的行
      }
    }

    // 第一个转入列内容 寻找可匹配的ID
    var secondTd = tds[thIndex];
    var comboIdElement = null;

    // 快手和头条分开处理，快手此处无类名
    if (!config[type].domClassName) {
      comboIdElement = secondTd;
    } else {
      comboIdElement = secondTd.querySelector(config[type].domClassName);
    }
    if (!comboIdElement) {
      continue; // 忽略没有目标类名的元素
    }

    var comboId = comboIdElement.textContent.trim();
    var numericValue = comboId.replace(/\D/g, ''); // 保留数字内容

    if (!whitelist.includes(numericValue)) {
      // 未匹配到，则第二个转出的列内容继续找 如果找不到则进行删除
      if (thIndex2 !== -1) {
        let secondTd2 = tds[thIndex2];
        var comboIdElement2 = null;
        // 快手和头条分开处理，快手此处无类名
        if (!config[type].domClassName) {
          comboIdElement2 = secondTd;
        } else {
          comboIdElement2 = secondTd2.querySelector(config[type].domClassName);
        }

        if (!comboIdElement2) {
          continue; // 忽略没有目标类名的元素
        }

        var comboId2 = comboIdElement2.textContent.trim();
        var numericValue2 = comboId2.replace(/\D/g, ''); // 保留数字内容

        if (!whitelist.includes(numericValue2)) {
          row.remove(); // 隐藏不在白名单内的行
        }
      } else {
        row.remove(); // 隐藏不在白名单内的行
      }
    }
  }
}

// 隐藏列
function hideColumnsAndCellsNotInWhitelist() {
  var thead = document.querySelector('thead');
  if (!thead) {
    console.log('未找到表格的 thead 元素');
    return;
  }

  var thList = thead.querySelectorAll('th');
  for (var i = 0; i < thList.length; i++) {
    var th = thList[i];

    var contentTitle = null;
    if (config[type].domTitleClassName) {
      contentTitle = th.querySelector(config[type].domTitleClassName);
    } else {
      contentTitle = th;
    }
    if (!contentTitle) {
      continue; // 忽略没有目标类名的元素
    }

    var content = contentTitle.textContent.trim();
    if (
      !config[type].needSaveColumnTitle.includes(content) &&
      !content.includes('总金额')
    ) {
      th.style.display = 'none'; // 隐藏不在白名单内的 th 标签

      var tbody = document.querySelector('tbody');
      if (!tbody) {
        console.log('未找到表格的 tbody 元素');
        return;
      }

      var rows = tbody.querySelectorAll('tr');
      for (var j = 0; j < rows.length; j++) {
        var tdList = rows[j].querySelectorAll('td');
        if (tdList.length > i) {
          tdList[i].style.display = 'none'; // 隐藏对应下标的 td 标签
        }
      }
    }
  }

  // 列头宽度优化
  var colgroupAll = document.querySelectorAll('colgroup');
  if (colgroupAll.length > 0) {
    colgroupAll.forEach((element) => {
      element.remove();
    });
  }

  if (type == '头条') {
    var tableAll = document.querySelectorAll('table');
    tableAll.forEach((ele) => {
      ele.style.minWidth = 'auto';
    });
  } else {
    // 自适应逻辑已在过滤阶段处理
    var tdAll = document.querySelectorAll('td');
	if(tdAll){
	    tdAll.forEach((tdTemp) => {
      		tdTemp.style.width = "25%%"
		});	
	}
    // end
  }
}
`),
			&ret),
			chromedp.Sleep(15*time.Second),
		); err != nil {
			log.Println(TaskMsg + "处理表格dom数据js错误" + err.Error())
			panic(TaskMsg + "处理表格dom数据js错误" + err.Error())
		}

		//双中判定，防止跳转到其他页面，空元素dom节点不仅仅是列表页面有
		var res1Count = -1
		chromedp.Run(ctx3, chromedp.Evaluate(`document.querySelectorAll(".agent-table-tbody tr").length`, &res1Count))
		if res1Count <= 0 {
			log.Println(TaskMsg + "表格内行数量" + strconv.Itoa(res1Count))
			panic(TaskMsg + "表格内行数量" + strconv.Itoa(res1Count))
		}

		var screenshot []byte
		if err := chromedp.Run(ctx3,
			//.agent-table-container
			chromedp.Screenshot(".agent-table-fixed-column", &screenshot, chromedp.ByQuery)); err != nil {
			log.Println(TaskMsg + "表格截图错误" + err.Error())
			panic(TaskMsg + "表格截图错误" + err.Error())
		}

		//创建日志文件夹
		utils2.CreateDir("../static/ks_screen_snap/")

		now := time.Now()
		screenshotImageName := DoTaskRequest.JobNo + "_" + strconv.Itoa(int(now.Unix())) + ".png"
		// 将截图保存到文件
		if err := os.WriteFile("../static/ks_screen_snap/"+screenshotImageName, screenshot, 0644); err != nil {
			log.Println(TaskMsg + "保存截图文件错误" + err.Error())
			panic(TaskMsg + "保存截图文件错误" + err.Error())
		}

		path, err := utils.UploadLocalFile("../static/ks_screen_snap/"+screenshotImageName, "data/ks/ks_screen_snap/"+screenshotImageName)
		if err != nil {
			log.Println(TaskMsg + "上传到tos错误" + err.Error())
			panic(TaskMsg + "上传到tos错误" + err.Error())
		}

		log.Println(TaskMsg+"截图保存成功,保存tos路径:", path)

		//修改任务状态
		database.CrmDB.Model(&model.CrmScreenshotJob{}).Where("job_no = ?", DoTaskRequest.JobNo).Updates(map[string]interface{}{"status": 2, "url": path, "fail_msg": ""})

		//删除截图文件
		err = os.Remove("../static/ks_screen_snap/" + screenshotImageName)
		if err != nil {
			log.Println(TaskMsg+"删除系统本地截图文件错误", err)
		}
		var requestData = make(map[string]interface{})
		requestData["job_no"] = DoTaskRequest.JobNo
		requestData["image_url"] = path
		requestData["notify_type"] = 1
		log.Println(TaskMsg+"请求php参数", requestData)
		//通知php给审批单发起人发送截图
		library.SendPOSTRequest("http://127.0.0.1:8000/external/screenshot/notify", requestData)

	} else {
		panic(TaskMsg + "表格元素dom节点数量" + strconv.Itoa(resCount))
	}

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
